import 'zone.js';
import { bootstrapApplication } from '@angular/platform-browser';
import { provideRouter } from '@angular/router';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { provideMarkdown } from 'ngx-markdown';
import { AppShellComponent } from './app/app-shell.component';
import { routes } from './app/app.routes';
import { authExpiryInterceptor } from './app/services/auth-expiry.interceptor';

bootstrapApplication(AppShellComponent, {
  providers: [
    provideRouter(routes),
    provideHttpClient(withInterceptors([authExpiryInterceptor])),
    provideMarkdown()
  ]
})
.catch((err) => console.error(err));
