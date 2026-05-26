import { Component, Input, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-code-block',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="code-block-container">
      <div class="code-block-header">
        <span class="language">{{ language || 'code' }}</span>
        <button
          class="copy-button"
          (click)="copyCode()"
          [class.copied]="copied">
          {{ copied ? 'Copied!' : 'Copy' }}
        </button>
      </div>
      <div class="code-content">
        <pre><code [class]="codeClass">{{ code }}</code></pre>
      </div>
    </div>
  `,
  styles: [`
    .code-block-container {
      background: #000;
      border-radius: 0.375rem;
      margin: 1rem 0;
      overflow: hidden;
    }

    .code-block-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      background: #1f2937;
      padding: 0.75rem 1rem;
      border-bottom: 1px solid #374151;
    }

    .language {
      color: #9ca3af;
      font-size: 0.875rem;
      font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    }

    .copy-button {
      background: transparent;
      border: 1px solid #4b5563;
      color: #9ca3af;
      padding: 0.25rem 0.75rem;
      border-radius: 0.25rem;
      font-size: 0.75rem;
      cursor: pointer;
      transition: all 0.2s;
    }

    .copy-button:hover {
      background: #374151;
      color: #ffffff;
    }

    .copy-button.copied {
      background: #059669;
      border-color: #059669;
      color: #ffffff;
    }

    .code-content {
      padding: 1rem;
      overflow-x: auto;
    }

    .code-content pre {
      margin: 0;
      padding: 0;
      background: transparent !important;
    }

    .code-content code {
      color: #e5e7eb;
      font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace !important;
      font-size: 0.875rem;
      line-height: 1.5;
      white-space: pre;
      display: block;
      background: transparent !important;
    }

    /* Language-specific styling */
    .hljs-keyword { color: #c678dd; }
    .hljs-string { color: #98c379; }
    .hljs-comment { color: #5c6370; }
    .hljs-number { color: #d19a66; }
    .hljs-function { color: #61dafb; }
  `]
})
export class CodeBlockComponent implements OnInit {
  @Input() code: string = '';
  @Input() language: string = '';

  copied = false;
  codeClass = '';

  ngOnInit() {
    this.codeClass = this.language ? `language-${this.language} hljs` : 'hljs';
  }

  copyCode() {
    navigator.clipboard.writeText(this.code).then(() => {
      this.copied = true;
      setTimeout(() => {
        this.copied = false;
      }, 2000);
    });
  }
}