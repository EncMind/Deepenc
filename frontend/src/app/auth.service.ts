import { Injectable, Injector, signal } from '@angular/core';
import { initializeApp } from 'firebase/app';
import {
  getAuth,
  signInWithEmailAndPassword,
  createUserWithEmailAndPassword,
  signOut,
  onAuthStateChanged,
  updateProfile,
  sendEmailVerification,
  sendPasswordResetEmail,
  signInWithPopup,
  GoogleAuthProvider,
  setPersistence,
  browserLocalPersistence,
  User as FirebaseUser,
  Auth
} from 'firebase/auth';
import { Observable, BehaviorSubject } from 'rxjs';
import { environment } from '../environments/environment';
import { ConfigService, FirebaseClientConfig } from './services/config.service';
import { BillingService } from './services/billing.service';

export interface User {
  id: string;
  email: string;
  displayName: string;
  photoURL: string;
  isActive: boolean;
  preferences: UserPreferences;
  createdAt: number;
  updatedAt: number;
  lastLoginAt: number;
}

export interface UserPreferences {
  defaultModels: {
    openai: string;
    anthropic: string;
    gemini: string;
  };
}

export interface AuthUser {
  uid: string;
  email: string | null;
  displayName: string | null;
  photoURL: string | null;
  emailVerified: boolean;
}

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private auth: Auth | null = null;
  private googleProvider: GoogleAuthProvider | null = null;
  private userSubject = new BehaviorSubject<AuthUser | null>(null);
  private profileSubject = new BehaviorSubject<User | null>(null);
  private _isRegistering = false;
  private firebaseInitialized = false;

  // Signals for reactive UI
  currentUser = signal<AuthUser | null>(null);
  userProfile = signal<User | null>(null);
  isAuthenticated = signal<boolean>(false);
  isLoading = signal<boolean>(true);

  private debug(...args: unknown[]): void {
    if (!environment.debugMode || !args.length) {
      return;
    }
    console.debug('[AuthService]', ...args);
  }

  constructor(private configService: ConfigService, private injector: Injector) {
    this.initializeAuth();
  }

  private async initializeAuth() {
    try {
      const firebaseConfig = await this.configService.waitForFirebaseConfig();

      if (!firebaseConfig || !this.configService.isFirebaseConfigValidSync(firebaseConfig)) {
        console.error('Invalid Firebase configuration received');
        this.isLoading.set(false);
        return;
      }

      // Initialize Firebase with dynamic config
      const app = initializeApp(firebaseConfig);
      this.auth = getAuth(app);
      this.firebaseInitialized = true;

      // Initialize Google Auth Provider
      this.googleProvider = new GoogleAuthProvider();
      this.googleProvider.addScope('email');
      this.googleProvider.addScope('profile');

      // Set Firebase persistence to local storage (Firebase maintains session state)
      await setPersistence(this.auth, browserLocalPersistence);

      // Listen to Firebase auth state changes (source of truth)
      onAuthStateChanged(this.auth, async (firebaseUser) => {
        this.debug('Auth state changed:', firebaseUser ? `User ${firebaseUser.email} logged in` : 'User logged out');
        
        if (firebaseUser) {
          try {
            // Check if user signed in with Google (OAuth providers bypass email verification)
            const isGoogleUser = firebaseUser.providerData.some(
              provider => provider.providerId === 'google.com'
            );

            // Check if email is verified for security (skip for Google users)
            if (!firebaseUser.emailVerified && !isGoogleUser) {
              this.debug('User email not verified, clearing auth state');
              this.clearUser();
              this.isLoading.set(false);
              return;
            }

            const authUser: AuthUser = {
              uid: firebaseUser.uid,
              email: firebaseUser.email,
              displayName: firebaseUser.displayName,
              photoURL: firebaseUser.photoURL,
              emailVerified: firebaseUser.emailVerified
            };

            this.currentUser.set(authUser);
            this.isAuthenticated.set(true);
            this.userSubject.next(authUser);
            this.debug('User authenticated successfully, setting auth state');

            // Only load user profile if we're not in the middle of registration
            if (!this._isRegistering) {
              try {
                await this.loadUserProfile();
              } catch (profileError) {
                console.error('Profile loading failed in Firebase auth handler:', profileError);
                setTimeout(() => {
                  this.loadUserProfile().catch((retryErr) => {
                    console.error('Profile reload retry failed:', retryErr);
                  });
                }, 2000);
              }
            }
          } catch (error) {
            console.error('Error processing authenticated user:', error);
            this.clearUser();
          }
        } else {
          this.debug('No authenticated user found, clearing auth state');
          this.clearUser();
        }
        
        this.isLoading.set(false);
        this.debug('Auth initialization complete. Authenticated:', this.isAuthenticated());
      });
      
    } catch (error) {
      console.error('Failed to initialize Firebase auth:', error);
      this.isLoading.set(false);
    }
  }

  private ensureFirebaseInitialized(): void {
    if (!this.firebaseInitialized || !this.auth || !this.googleProvider) {
      throw new Error('Firebase not yet initialized. Please wait for configuration to load.');
    }
  }

  private clearUser() {
    this.currentUser.set(null);
    this.userProfile.set(null);
    this.isAuthenticated.set(false);
    this.userSubject.next(null);
    this.profileSubject.next(null);
    this.clearStoredUser();

    // Centralize billing cache clearing so all logout paths are covered.
    // Using Injector to avoid circular dependency (BillingService also depends on AuthService).
    try {
      const billingService = this.injector.get(BillingService, null);
      billingService?.clearCachedSubscriptionInfo();
      billingService?.clearCachedUsageStats();
    } catch (error) {
      this.debug('BillingService not available during logout cache clear');
    }
  }

  private clearStoredUser(): void {
    try {
      localStorage.removeItem('deepenc_user');

      localStorage.removeItem('deepenc_anonymous_user_id');
      localStorage.removeItem('deepenc-dismissed-alerts');

      // If session keys need clearing in future, remove specific keys instead of clearing all
      // sessionStorage.removeItem('deepenc_specific_key');

      this.debug('All stored user credentials and app data cleared');
    } catch (error) {
      console.error('Failed to clear stored user:', error);
    }
  }

  get user$(): Observable<AuthUser | null> {
    return this.userSubject.asObservable();
  }

  get profile$(): Observable<User | null> {
    return this.profileSubject.asObservable();
  }

  async register(email: string, password: string, displayName: string): Promise<{ requiresEmailVerification: boolean; authUser?: AuthUser }> {
    try {
      this.ensureFirebaseInitialized();
      this._isRegistering = true;
      
      // Create user in Firebase first
      const credential = await createUserWithEmailAndPassword(this.auth, email, password);
      
      // Update display name
      if (credential.user && displayName) {
        await updateProfile(credential.user, { displayName });
      }

      // Send email verification with custom settings
      try {
        await sendEmailVerification(credential.user, {
          url: window.location.origin + '/auth', // Redirect back to auth page
        });
        this.debug('Verification email sent successfully to:', credential.user.email);
      } catch (emailError) {
        console.error('Failed to send verification email:', emailError);
        // Don't throw here - user is created, just email sending failed
      }

      // Return early - user needs to verify email before account is fully created
      return {
        requiresEmailVerification: true,
        authUser: {
          uid: credential.user.uid,
          email: credential.user.email,
          displayName: credential.user.displayName,
          photoURL: credential.user.photoURL,
          emailVerified: credential.user.emailVerified,
        }
      };
    } catch (error: any) {
      console.error('Registration error:', error);
      throw new Error(this.getFirebaseErrorMessage(error.code) || 'Registration failed');
    } finally {
      this._isRegistering = false;
    }
  }

  async resendVerificationEmail(): Promise<void> {
    try {
      const user = this.auth.currentUser;
      if (!user) {
        throw new Error('No user found');
      }

      await sendEmailVerification(user, {
        url: window.location.origin + '/auth',
      });
      this.debug('Verification email resent to:', user.email);
    } catch (error: any) {
      console.error('Failed to resend verification email:', error);
      throw new Error('Failed to resend verification email');
    }
  }

  async completeRegistration(): Promise<AuthUser> {
    try {
      const user = this.auth.currentUser;
      if (!user) {
        throw new Error('No user found');
      }

      // Reload user to get latest emailVerified status
      await user.reload();

      if (!user.emailVerified) {
        throw new Error('Email not yet verified');
      }

      // Get the Firebase ID token
      const idToken = await user.getIdToken();

      // Now register with backend using the Firebase token
      const response = await fetch('/api/auth/register', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${idToken}`
        },
        body: JSON.stringify({
          email: user.email,
          displayName: user.displayName || ''
        })
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }

      const authUser: AuthUser = {
        uid: user.uid,
        email: user.email,
        displayName: user.displayName,
        photoURL: user.photoURL,
        emailVerified: user.emailVerified
      };

      // Load the user profile that was just created
      await this.loadUserProfile();

      return authUser;
    } catch (error: any) {
      console.error('Complete registration error:', error);
      throw new Error(error.message || 'Failed to complete registration');
    }
  }

  async login(email: string, password: string): Promise<AuthUser> {
    try {
      this.ensureFirebaseInitialized();
      // Ensure persistence is set before login
      await setPersistence(this.auth!, browserLocalPersistence);
      this.debug('Persistence confirmed before login');
      
      const credential = await signInWithEmailAndPassword(this.auth, email, password);
      this.debug('Login successful, user:', credential.user.email);
      const authUser: AuthUser = {
        uid: credential.user.uid,
        email: credential.user.email,
        displayName: credential.user.displayName,
        photoURL: credential.user.photoURL,
        emailVerified: credential.user.emailVerified
      };

      this.currentUser.set(authUser);
      this.isAuthenticated.set(true);
      this.userSubject.next(authUser);

      // Load user profile after successful login
      try {
        this.debug('Loading user profile after login');
        await this.loadUserProfile();
      } catch (error) {
        console.error('Failed to load profile after login:', error);
      }

      this.debug('Returning auth user from login:', authUser.email);
      return authUser;
    } catch (error: any) {
      console.error('Login error:', error);
      throw new Error(this.getFirebaseErrorMessage(error.code) || 'Login failed');
    }
  }

  async signInWithGoogle(): Promise<AuthUser> {
    try {
      this.ensureFirebaseInitialized();
      const result = await signInWithPopup(this.auth!, this.googleProvider!);
      const authUser: AuthUser = {
        uid: result.user.uid,
        email: result.user.email,
        displayName: result.user.displayName,
        photoURL: result.user.photoURL,
        emailVerified: result.user.emailVerified
      };

      this.currentUser.set(authUser);
      this.isAuthenticated.set(true);
      this.userSubject.next(authUser);

      // For Google sign-in, the email is automatically verified
      // Always ensure user has a profile in Cosmos DB
      try {
        // Try to load existing profile first
        await this.loadUserProfile();
        
        // If no profile exists, loadUserProfile will automatically create one
        // via the 404 fallback logic since Google users have verified emails
      } catch (error) {
        console.error('Failed to ensure user profile for Google user:', error);
      }

      this.debug('Google sign-in successful:', authUser.email);
      return authUser;
    } catch (error: any) {
      console.error('Google sign-in error:', error);
      throw new Error(this.getFirebaseErrorMessage(error.code) || 'Google sign-in failed');
    }
  }

  async logout(): Promise<void> {
    try {
      this.debug('Logging out user...');
      
      // Sign out from Firebase first so session tokens are cleared
      if (this.auth) {
        await signOut(this.auth);
      }
      
      // Then clear local state and any cached data
      this.clearUser();
      
      this.debug('Logout successful');
    } catch (error) {
      console.error('Logout error:', error);
      this.clearUser();
      throw error;
    }
  }

  async sendPasswordReset(email: string): Promise<void> {
    try {
      this.ensureFirebaseInitialized();
      await sendPasswordResetEmail(this.auth!, email, {
        url: window.location.origin + '/auth', // Redirect back to auth page after reset
      });
      this.debug('Password reset email sent to:', email);
    } catch (error: any) {
      console.error('Password reset error:', error);
      throw new Error(this.getFirebaseErrorMessage(error.code) || 'Failed to send password reset email');
    }
  }

  async getIdToken(): Promise<string | null> {
    if (!this.auth) {
      return null;
    }
    const user = this.auth.currentUser;
    this.debug('[Debug] Firebase current user:', user ? `${user.email} (verified: ${user.emailVerified})` : 'None');
    this.debug('[Debug] Auth service authenticated signal:', this.isAuthenticated());
    this.debug('[Debug] Auth service current user signal:', this.currentUser()?.email || 'None');
    
    if (user) {
      try {
        const token = await user.getIdToken();
        this.debug('[Debug] Token successfully retrieved, length:', token?.length || 0);
        return token;
      } catch (error) {
        console.error('[Debug] Error getting ID token:', error);
        return null;
      }
    }
    
    this.debug('[Debug] No current user, returning null token');
    return null;
  }


  private async loadUserProfile(): Promise<void> {
    try {
      this.debug('Loading user profile...');
      const token = await this.getIdToken();
      if (!token) {
        this.debug('No token available for profile loading');
        return;
      }

      this.debug('Making request to /api/auth/profile');
      const response = await fetch('/api/auth/profile', {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      });

      this.debug('Profile API response status:', response.status);
      
      if (response.ok) {
        const profile: User = await response.json();
        this.debug('Profile loaded successfully:', profile.email);
        this.userProfile.set(profile);
        this.profileSubject.next(profile);
      } else if (response.status === 404) {
        // User not found in Cosmos DB - try to create profile if email is verified
        const currentUser = this.currentUser();
        this.debug('Profile not found (404), current user:', currentUser?.email, 'verified:', currentUser?.emailVerified);
        if (currentUser?.emailVerified) {
          this.debug('User profile not found, attempting to create profile for:', currentUser.email);
          try {
            await this.createUserProfile(currentUser);
            this.debug('Profile creation completed, profile should now be loaded');
          } catch (error) {
            console.error('Failed to auto-create user profile:', error);
            // Set profile to null to keep showing loading state
            this.userProfile.set(null);
            this.profileSubject.next(null);
          }
        } else {
          this.debug('User email not verified, cannot create profile');
          this.userProfile.set(null);
          this.profileSubject.next(null);
        }
      } else {
        const errorText = await response.text();
        console.error('Failed to load profile:', response.status, errorText);
        // Set profile to null to keep showing loading state on error
        this.userProfile.set(null);
        this.profileSubject.next(null);
      }
    } catch (error) {
      console.error('Error loading user profile:', error);
      // Set profile to null to keep showing loading state on network error
      this.userProfile.set(null);
      this.profileSubject.next(null);
    }
  }

  private async createUserProfile(authUser: AuthUser): Promise<void> {
    this.debug('Creating user profile for:', authUser.email);
    const token = await this.getIdToken();
    if (!token) {
      throw new Error('Not authenticated');
    }
    const response = await fetch('/api/auth/register', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({
        email: authUser.email,
        displayName: authUser.displayName || ''
      })
    });

    this.debug('Profile creation response status:', response.status);
    
    if (!response.ok) {
      const error = await response.text();
      console.error('Profile creation failed:', error);
      throw new Error(error);
    }

    this.debug('Profile created successfully, reloading...');
    // Load the newly created profile
    await this.loadUserProfile();
  }

  async updateProfile(updates: {
    displayName?: string;
    photoURL?: string;
    preferences?: UserPreferences;
  }): Promise<User> {
    try {
      const token = await this.getIdToken();
      if (!token) throw new Error('Not authenticated');

      const response = await fetch('/api/auth/profile', {
        method: 'PUT',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(updates)
      });

      if (!response.ok) {
        throw new Error('Failed to update profile');
      }

      const updatedProfile: User = await response.json();
      this.userProfile.set(updatedProfile);
      this.profileSubject.next(updatedProfile);

      // Update Firebase profile if needed
      const firebaseUser = this.auth.currentUser;
      if (firebaseUser && (updates.displayName || updates.photoURL)) {
        await updateProfile(firebaseUser, {
          displayName: updates.displayName || firebaseUser.displayName,
          photoURL: updates.photoURL || firebaseUser.photoURL
        });
      }

      return updatedProfile;
    } catch (error: any) {
      console.error('Profile update error:', error);
      throw new Error(error.message || 'Failed to update profile');
    }
  }

  async deleteAccount(): Promise<void> {
    try {
      const token = await this.getIdToken();
      if (!token) throw new Error('Not authenticated');

      // Delete from backend first
      const response = await fetch('/api/auth/profile', {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      });

      if (!response.ok) {
        throw new Error('Failed to delete account');
      }

      // Firebase user deletion is handled by the backend
    } catch (error: any) {
      console.error('Account deletion error:', error);
      throw new Error(error.message || 'Failed to delete account');
    }
  }

  getCurrentUser(): AuthUser | null {
    return this.currentUser();
  }

  getUserProfile(): User | null {
    return this.userProfile();
  }

  isLoggedIn(): boolean {
    return this.isAuthenticated();
  }

  private getFirebaseErrorMessage(errorCode: string): string {
    switch (errorCode) {
      case 'auth/invalid-credential':
      case 'auth/wrong-password':
        return 'Invalid password. Please check your password and try again.';
      
      case 'auth/user-not-found':
        return 'No account found with this email address.';
      
      case 'auth/email-already-in-use':
        return 'An account with this email address already exists.';
      
      case 'auth/weak-password':
        return 'Password is too weak. Please choose a stronger password.';
      
      case 'auth/invalid-email':
        return 'Please enter a valid email address.';
      
      case 'auth/user-disabled':
        return 'This account has been disabled. Please contact support.';
      
      case 'auth/too-many-requests':
        return 'Too many failed attempts. Please try again later.';
      
      case 'auth/network-request-failed':
        return 'Network error. Please check your connection and try again.';
      
      case 'auth/popup-closed-by-user':
        return 'Sign-in was cancelled.';
      
      case 'auth/cancelled-popup-request':
        return 'Only one sign-in popup allowed at a time.';
      
      case 'auth/popup-blocked':
        return 'Sign-in popup was blocked by the browser.';
      
      case 'auth/invalid-verification-code':
        return 'Invalid verification code.';
      
      case 'auth/invalid-verification-id':
        return 'Invalid verification ID.';
      
      case 'auth/missing-verification-code':
        return 'Please enter the verification code.';
      
      case 'auth/missing-verification-id':
        return 'Missing verification ID.';
      
      case 'auth/code-expired':
        return 'The verification code has expired.';
      
      case 'auth/invalid-phone-number':
        return 'Please enter a valid phone number.';
      
      case 'auth/missing-phone-number':
        return 'Please enter a phone number.';
      
      case 'auth/requires-recent-login':
        return 'Please sign out and sign in again to complete this action.';
      
      case 'auth/credential-already-in-use':
        return 'This credential is already associated with another account.';
      
      case 'auth/custom-token-mismatch':
        return 'Invalid authentication token.';
      
      case 'auth/invalid-custom-token':
        return 'Invalid authentication token format.';
      
      case 'auth/invalid-user-token':
        return 'Your session has expired. Please sign in again.';
      
      case 'auth/user-token-expired':
        return 'Your session has expired. Please sign in again.';
      
      default:
        return 'An authentication error occurred. Please try again.';
    }
  }
}
