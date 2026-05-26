import { Component, Input, OnInit, OnChanges, SimpleChanges } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MarkdownModule } from 'ngx-markdown';

import { BlockParserService, TextBlock } from '../services/block-parser.service';
import { CodeBlockComponent } from './code-block.component';

@Component({
  selector: 'app-blocks-renderer',
  standalone: true,
  imports: [CommonModule, MarkdownModule, CodeBlockComponent],
  template: `
    <div class="blocks-container">
      <ng-container *ngFor="let block of blocks; trackBy: trackByIndex">

        <!-- Markdown Block -->
        <div *ngIf="block.type === 'markdown'" class="markdown-block">
          <markdown
            [data]="block.content"
            [options]="markdownOptions"
            ngPreserveWhitespaces>
          </markdown>
        </div>

        <!-- Code Block -->
        <app-code-block
          *ngIf="block.type === 'code'"
          [code]="block.content"
          [language]="getEffectiveLanguage(block)">
        </app-code-block>

      </ng-container>
    </div>
  `,
  styles: [`
    .blocks-container {
      display: flex;
      flex-direction: column;
      gap: 1rem;
    }

    .markdown-block {
      /* Ensure markdown renders properly */
    }

    /* Override any conflicting global styles for markdown within blocks */
    .markdown-block markdown {
      color: inherit;
    }

    .markdown-block markdown p {
      margin-bottom: 1rem;
    }

    .markdown-block markdown pre,
    .markdown-block markdown code {
      /* Prevent conflicts with our custom code blocks */
      display: none !important;
    }
  `]
})
export class BlocksRendererComponent implements OnInit, OnChanges {
  @Input() content: string = '';
  @Input() enableCodeBlocks: boolean = true;

  blocks: TextBlock[] = [];
  markdownOptions = {
    breaks: true
  };

  constructor(private blockParser: BlockParserService) {}

  ngOnInit() {
    this.parseContent();
  }

  ngOnChanges(changes: SimpleChanges) {
    if (changes['content']) {
      this.parseContent();
    }
  }

  private parseContent() {
    if (!this.content) {
      this.blocks = [];
      return;
    }

    if (this.enableCodeBlocks) {
      this.blocks = this.blockParser.parseTextIntoBlocks(this.content);
    } else {
      // Fallback to single markdown block
      this.blocks = [{
        type: 'markdown',
        content: this.content
      }];
    }
  }

  getEffectiveLanguage(block: TextBlock): string {
    if (block.language && block.language !== 'plaintext') {
      return block.language;
    }
    return this.blockParser.detectLanguage(block.content);
  }

  trackByIndex(index: number, block: TextBlock): any {
    return `${block.type}-${index}`;
  }
}