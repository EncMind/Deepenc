import { Routes } from '@angular/router';
import { AuthGuard, UnauthGuard } from './auth.guard';
import { Injectable } from '@angular/core';
import { CanActivate, Router } from '@angular/router';
import { Observable, timer } from 'rxjs';
import { filter, take, map } from 'rxjs/operators';
import { AuthService } from './auth.service';

@Injectable({
  providedIn: 'root'
})
export class HomeGuard implements CanActivate {
  constructor(private authService: AuthService, private router: Router) {}

  canActivate(): Observable<boolean> {
    // Poll until loading is complete, then check auth state
    return timer(0, 100).pipe(
      filter(() => !this.authService.isLoading()),
      take(1),
      map(() => {
        const user = this.authService.getCurrentUser();
        
        if (user && user.emailVerified) {
          // Authenticated user should go to full app
          this.router.navigate(['/app']);
          return false;
        } else {
          // Unauthenticated user can use free chat
          return true;
        }
      })
    );
  }
}

export const routes: Routes = [
  {
    path: '',
    loadComponent: () => import('./free-chat/free-chat.component').then(m => m.FreeChatComponent),
    canActivate: [HomeGuard]
  },
  {
    path: 'about',
    loadComponent: () => import('./about/about.component').then(m => m.AboutComponent)
  },
  {
    path: 'app',
    loadComponent: () => import('./app.component').then(m => m.AppComponent),
    canActivate: [AuthGuard]
  },
  {
    path: 'auth',
    loadComponent: () => import('./auth/auth.component').then(m => m.AuthComponent),
    canActivate: [UnauthGuard]
  },
  {
    path: 'account',
    loadComponent: () => import('./profile/profile.component').then(m => m.ProfileComponent),
    canActivate: [AuthGuard]
  },
  {
    path: 'pricing',
    loadComponent: () => import('./pricing/pricing.component').then(m => m.PricingComponent)
  },
  {
    path: 'usage',
    loadComponent: () => import('./usage/usage-dashboard.component').then(m => m.UsageDashboardComponent),
    canActivate: [AuthGuard]
  },
  {
    path: 'terms',
    loadComponent: () => import('./legal/terms.component').then(m => m.TermsComponent)
  },
  {
    path: 'privacy',
    loadComponent: () => import('./legal/privacy.component').then(m => m.PrivacyComponent)
  },
  {
    path: '**',
    redirectTo: ''
  }
];