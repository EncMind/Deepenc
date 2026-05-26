import { Injectable } from '@angular/core';
import { CanActivate, Router } from '@angular/router';
import { map, take, filter, switchMap } from 'rxjs/operators';
import { Observable, combineLatest, timer } from 'rxjs';
import { AuthService } from './auth.service';

@Injectable({
  providedIn: 'root'
})
export class AuthGuard implements CanActivate {
  constructor(private authService: AuthService, private router: Router) {}

  canActivate(): Observable<boolean> {
    // Poll until loading is complete, then check auth state
    return timer(0, 100).pipe(
      filter(() => {
        const isLoading = this.authService.isLoading();
        return !isLoading;
      }),
      take(1),
      map(() => {
        const user = this.authService.getCurrentUser();
        
        if (user && user.emailVerified) {
          return true;
        } else {
          this.router.navigate(['/auth']);
          return false;
        }
      })
    );
  }
}

@Injectable({
  providedIn: 'root'
})
export class UnauthGuard implements CanActivate {
  constructor(private authService: AuthService, private router: Router) {}

  canActivate(): Observable<boolean> {
    // Poll until loading is complete, then check auth state
    return timer(0, 100).pipe(
      filter(() => {
        const isLoading = this.authService.isLoading();
        return !isLoading;
      }),
      take(1),
      map(() => {
        const user = this.authService.getCurrentUser();
        
        if (!user) {
          return true;
        } else {
          this.router.navigate(['/app']);
          return false;
        }
      })
    );
  }
}
