import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';
import { AuthService } from '../auth.service';

// Intercepts 401s to prompt the user to re-authenticate.
export const authExpiryInterceptor: HttpInterceptorFn = (req, next) => {
  const authService = inject(AuthService);
  const router = inject(Router);

  return next(req).pipe(
    catchError((error) => {
      if (error?.status === 401) {
        authService.logout().catch(() => {
          /* ignore logout errors during expiry handling */
        });
        router.navigate(['/auth']);
      }
      return throwError(() => error);
    })
  );
};
