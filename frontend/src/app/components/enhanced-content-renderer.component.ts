import { Component, Input, OnInit, OnChanges, SimpleChanges } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MarkdownModule } from 'ngx-markdown';
import {
  EnhancedMessage,
  ParsedContent,
  ContentBlock,
  ContentRenderingStrategy,
  ContentRenderResult,
  ParsedContentAnalyzer
} from '../models/content.models';
import { ContentParserService } from '../services/content-parser.service';
import { MarkdownConfigService } from '../services/markdown-config.service';
import { MessageFormatterService } from '../services/message-formatter.service';
import { environment } from '../../environments/environment';

@Component({
  selector: 'app-enhanced-content-renderer',
  standalone: true,
  imports: [CommonModule, MarkdownModule],
  template: `
    <div class="enhanced-content-renderer" [attr.data-strategy]="currentStrategy">
      <!-- Loading state -->
      <div *ngIf="isLoading" class="loading-indicator">
        <span>Processing content...</span>
      </div>

      <!-- Error state -->
      <div *ngIf="hasError" class="error-indicator">
        <span>Failed to process content. Falling back to basic rendering.</span>
      </div>

      <!-- Rendered content based on strategy -->
      <ng-container [ngSwitch]="renderResult?.component">
        <!-- Blocks-based rendering for structured content -->
        <div *ngSwitchCase="'blocks'" class="blocks-renderer">
          <div *ngFor="let block of renderResult.data.blocks; trackBy: trackBlock"
               class="content-block"
               [attr.data-block-type]="block.type"
               [attr.data-confidence]="block.confidence">

            <!-- Code blocks -->
            <div *ngIf="block.type === 'code'" class="code-block">
              <div class="code-header" *ngIf="block.language">
                <span class="language-tag">{{ block.language }}</span>
                <span class="confidence-indicator"
                      [class.high-confidence]="block.confidence > 0.9"
                      [class.medium-confidence]="block.confidence > 0.7 && block.confidence <= 0.9"
                      [class.low-confidence]="block.confidence <= 0.7">
                  {{ formatConfidence(block.confidence) }}
                </span>
              </div>
              <pre><code [class]="'language-' + (block.language || 'text')">{{ block.content }}</code></pre>
            </div>

            <!-- Math blocks -->
            <div *ngIf="block.type === 'math'" class="math-block">
              <div class="math-content" [innerHTML]="renderMath(block.content)"></div>
            </div>

            <!-- Thinking blocks (Claude-specific) -->
            <div *ngIf="block.type === 'thinking'" class="thinking-block">
              <details>
                <summary>💭 Thinking process</summary>
                <div class="thinking-content">
                  <markdown [data]="sanitizeContent(block.content)" [options]="markdownOptions"></markdown>
                </div>
              </details>
            </div>

            <!-- List blocks -->
            <div *ngIf="block.type === 'list'" class="list-block">
              <markdown [data]="sanitizeContent(block.content)" [options]="markdownOptions"></markdown>
            </div>

            <!-- Text blocks -->
            <div *ngIf="block.type === 'text'" class="text-block">
              <markdown [data]="sanitizeContent(block.content)" [options]="markdownOptions"></markdown>
            </div>

            <!-- Unknown block types -->
            <div *ngIf="!['code', 'math', 'thinking', 'list', 'text'].includes(block.type)"
                 class="unknown-block">
              <div class="block-type-label">{{ formatBlockType(block.type) }}</div>
              <markdown [data]="sanitizeContent(block.content)" [options]="markdownOptions"></markdown>
            </div>
          </div>

          <!-- Parsing metadata -->
          <div *ngIf="showMetadata && renderResult.data.metadata" class="parsing-metadata">
            <details>
              <summary>🔍 Parsing Information</summary>
              <div class="metadata-content">
                <p><strong>Provider:</strong> {{ formatProvider(renderResult.data.provider) }}</p>
                <p><strong>Confidence:</strong> {{ formatConfidence(renderResult.data.confidence) }}</p>
                <p><strong>Blocks:</strong> {{ renderResult.data.blocks.length }}</p>
                <p><strong>Strategy:</strong> {{ renderResult.renderingStrategy }}</p>
                <p><strong>Parsed:</strong> {{ formatTimestamp(renderResult.data.parsedAt) }}</p>
              </div>
            </details>
          </div>
        </div>

        <!-- Hybrid rendering (blocks + markdown) -->
        <div *ngSwitchCase="'hybrid'" class="hybrid-renderer">
          <div class="structured-content">
            <!-- Render high-confidence blocks first -->
            <div *ngFor="let block of getHighConfidenceBlocks()" class="priority-block">
              <!-- Same block rendering as above -->
              <div [ngSwitch]="block.type">
                <div *ngSwitchCase="'code'" class="code-block">
                  <pre><code [class]="'language-' + (block.language || 'text')">{{ block.content }}</code></pre>
                </div>
                <div *ngSwitchDefault class="other-block">
                  <markdown [data]="sanitizeContent(block.content)" [options]="markdownOptions"></markdown>
                </div>
              </div>
            </div>
          </div>

          <!-- Render remaining content as markdown -->
          <div class="markdown-content">
            <markdown [data]="sanitizeContent(getRemainingContent())" [options]="markdownOptions"></markdown>
          </div>
        </div>

        <!-- Fallback markdown rendering -->
        <div *ngSwitchDefault class="markdown-renderer">
          <markdown [data]="sanitizeContent(message.content)" [options]="markdownOptions"></markdown>
        </div>
      </ng-container>

      <!-- Debug information (development only) -->
      <div *ngIf="showDebugInfo" class="debug-info">
        <details>
          <summary>🛠️ Debug Information</summary>
          <pre>{{ getDebugInfo() | json }}</pre>
        </details>
      </div>
    </div>
  `,
  styles: [`
    .enhanced-content-renderer {
      width: 100%;
    }

    .loading-indicator, .error-indicator {
      padding: 8px 12px;
      border-radius: 4px;
      margin-bottom: 8px;
      font-size: 0.9em;
    }

    .loading-indicator {
      background-color: #f0f8ff;
      color: #0066cc;
      border: 1px solid #b3d9ff;
    }

    .error-indicator {
      background-color: #fff5f5;
      color: #cc0000;
      border: 1px solid #ffb3b3;
    }

    .content-block {
      margin-bottom: 16px;
    }

    .code-block {
      background-color: #f8f9fa;
      border: 1px solid #e9ecef;
      border-radius: 6px;
      overflow: hidden;
    }

    .code-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 8px 12px;
      background-color: #e9ecef;
      border-bottom: 1px solid #dee2e6;
      font-size: 0.85em;
    }

    .language-tag {
      font-family: 'Monaco', 'Consolas', monospace;
      color: #495057;
      font-weight: 500;
    }

    .confidence-indicator {
      padding: 2px 6px;
      border-radius: 12px;
      font-size: 0.8em;
      font-weight: 600;
    }

    .high-confidence {
      background-color: #d4edda;
      color: #155724;
    }

    .medium-confidence {
      background-color: #fff3cd;
      color: #856404;
    }

    .low-confidence {
      background-color: #f8d7da;
      color: #721c24;
    }

    .code-block pre {
      margin: 0;
      padding: 12px;
      background-color: #f8f9fa;
      overflow-x: auto;
    }

    .code-block code {
      font-family: 'Monaco', 'Consolas', monospace;
      font-size: 0.9em;
      line-height: 1.4;
    }

    .math-block {
      padding: 16px;
      text-align: center;
      background-color: #fafafa;
      border-left: 4px solid #007acc;
      margin: 12px 0;
    }

    .thinking-block {
      margin: 12px 0;
    }

    .thinking-block summary {
      cursor: pointer;
      padding: 8px 12px;
      background-color: #f0f8ff;
      border: 1px solid #b3d9ff;
      border-radius: 4px;
      color: #0066cc;
      font-weight: 500;
    }

    .thinking-content {
      padding: 12px;
      background-color: #fafafa;
      border: 1px solid #e0e0e0;
      border-top: none;
      border-radius: 0 0 4px 4px;
    }

    .text-block {
      line-height: 1.6;
    }

    .list-block {
      margin: 8px 0;
    }

    .unknown-block {
      border: 1px dashed #ccc;
      padding: 12px;
      border-radius: 4px;
      background-color: #f9f9f9;
    }

    .block-type-label {
      font-size: 0.8em;
      color: #666;
      margin-bottom: 8px;
      font-weight: 500;
    }

    .parsing-metadata {
      margin-top: 16px;
      padding-top: 16px;
      border-top: 1px solid #eee;
    }

    .parsing-metadata summary {
      cursor: pointer;
      color: #666;
      font-size: 0.9em;
    }

    .metadata-content {
      padding: 12px 0;
      font-size: 0.85em;
      color: #666;
    }

    .metadata-content p {
      margin: 4px 0;
    }

    .hybrid-renderer .structured-content {
      margin-bottom: 16px;
    }

    .priority-block {
      margin-bottom: 12px;
    }

    .debug-info {
      margin-top: 16px;
      padding: 12px;
      background-color: #f0f0f0;
      border-radius: 4px;
      font-size: 0.8em;
    }

    .debug-info pre {
      max-height: 200px;
      overflow-y: auto;
      background-color: white;
      padding: 8px;
      border-radius: 2px;
    }
  `]
})
export class EnhancedContentRendererComponent implements OnInit, OnChanges {
  @Input() message!: EnhancedMessage;
  @Input() showMetadata = false;
  @Input() showDebugInfo = false;

  renderResult: ContentRenderResult | null = null;
  currentStrategy = 'loading';
  isLoading = false;
  hasError = false;
  markdownOptions: any;
  private readonly debugEnabled = environment.debugMode;

  private strategies: ContentRenderingStrategy[] = [];

  constructor(
    private contentParserService: ContentParserService,
    private markdownConfig: MarkdownConfigService,
    private messageFormatter: MessageFormatterService
  ) {
    this.markdownOptions = this.markdownConfig.getOptions();
    this.initializeStrategies();
  }

  private debugLog(...args: unknown[]): void {
    if (!this.debugEnabled) return;
    if (args.length === 0) return;
    console.debug(...args);
  }

  ngOnInit() {
    this.renderContent();
  }

  ngOnChanges(changes: SimpleChanges) {
    if (changes['message']) {
      this.renderContent();
    }
  }

  private initializeStrategies() {
    this.strategies = [
      new BlocksRenderingStrategy(),
      new HybridRenderingStrategy(),
      new MarkdownRenderingStrategy()
    ];
  }

  private async renderContent() {
    this.debugLog('🎨 [EnhancedRenderer] Starting renderContent()');
    this.debugLog('🎨 [EnhancedRenderer] Message content length:', this.message?.content?.length || 0);
    this.debugLog('🎨 [EnhancedRenderer] Message provider:', this.message?.provider);
    this.debugLog('🎨 [EnhancedRenderer] Has parsed content:', !!this.message?.parsedContent);

    if (!this.message?.content) {
      this.debugLog('🎨 [EnhancedRenderer] No content found, returning null');
      this.renderResult = null;
      return;
    }

    // Avoid logging actual content to protect privacy; log length only
    this.debugLog('🎨 [EnhancedRenderer] Content length:', this.message.content.length);

    this.isLoading = true;
    this.hasError = false;

    try {
      if (this.message.parsedContent) {
        this.debugLog('🎨 [EnhancedRenderer] Using existing parsed content');
        this.debugLog('🎨 [EnhancedRenderer] Parsed content blocks:', this.message.parsedContent.blocks?.length || 0);
        this.processWithParsedContent(this.message.parsedContent);
      } else {
        this.debugLog('🎨 [EnhancedRenderer] Parsing content with fallback');
        this.contentParserService.parseWithFallback(this.message.content, this.message.provider)
          .subscribe({
            next: (parsed) => {
              this.debugLog('🎨 [EnhancedRenderer] Content parsing completed');
              this.debugLog('🎨 [EnhancedRenderer] Parsed blocks:', parsed?.blocks?.length || 0);
              this.debugLog('🎨 [EnhancedRenderer] Parser confidence:', parsed?.confidence);
              this.processWithParsedContent(parsed);
            },
            error: (error) => {
              console.error('🎨 [EnhancedRenderer] Content parsing failed:', error);
              this.fallbackToMarkdown();
            }
          });
      }
    } catch (error) {
      console.error('🎨 [EnhancedRenderer] Content rendering failed:', error);
      this.fallbackToMarkdown();
    }
  }

  private processWithParsedContent(parsed: ParsedContent) {
    this.debugLog('🎨 [EnhancedRenderer] Processing parsed content');
    this.debugLog('🎨 [EnhancedRenderer] Parsed content provider:', parsed?.provider);
    this.debugLog('🎨 [EnhancedRenderer] Parsed content confidence:', parsed?.confidence);
    this.debugLog('🎨 [EnhancedRenderer] Available strategies:', this.strategies.length);

    // Test each strategy and log its decision
    if (this.debugEnabled) {
      this.strategies.forEach(s => {
        const canHandle = s.canHandle(this.message);
        const strategyName = s.constructor.name;
        this.debugLog(`🎨 [EnhancedRenderer] Strategy ${strategyName} canHandle: ${canHandle}`);
      });
    }

    const strategy = this.strategies.find(s => s.canHandle(this.message));

    if (strategy) {
      this.debugLog('🎨 [EnhancedRenderer] Selected strategy:', strategy.constructor.name);
      this.renderResult = strategy.render(this.message);
      this.renderResult.data = { ...parsed, message: this.message };
      this.currentStrategy = this.renderResult.renderingStrategy;

      this.debugLog('🎨 [EnhancedRenderer] Render result component:', this.renderResult.component);
      this.debugLog('🎨 [EnhancedRenderer] Render result confidence:', this.renderResult.confidence);
      this.debugLog('🎨 [EnhancedRenderer] Current strategy:', this.currentStrategy);

      // Log content blocks if available
      if (this.renderResult.data?.blocks) {
        this.debugLog('🎨 [EnhancedRenderer] Render result blocks:', this.renderResult.data.blocks.length);
        this.renderResult.data.blocks.forEach((block: any, index: number) => {
          this.debugLog(`🎨 [EnhancedRenderer] Block ${index}: type=${block.type}, confidence=${block.confidence}, content_length=${block.content?.length || 0}`);
        });
      }
    } else {
      this.debugLog('🎨 [EnhancedRenderer] No strategy found, falling back to markdown');
      this.fallbackToMarkdown();
    }

    this.isLoading = false;
    this.debugLog('🎨 [EnhancedRenderer] Processing completed, isLoading set to false');
  }

  private fallbackToMarkdown() {
    this.debugLog('🎨 [EnhancedRenderer] Falling back to markdown rendering');
    this.debugLog('🎨 [EnhancedRenderer] Content length for fallback:', this.message.content?.length || 0);

    this.renderResult = {
      component: 'markdown',
      data: { content: this.message.content },
      confidence: 0.5,
      renderingStrategy: 'markdown_fallback'
    };
    this.currentStrategy = 'markdown';
    this.hasError = true;
    this.isLoading = false;

    this.debugLog('🎨 [EnhancedRenderer] Fallback completed - using markdown renderer');
  }

  // Template helper methods
  trackBlock(index: number, block: ContentBlock): string {
    return `${block.type}-${index}-${block.content.slice(0, 20)}`;
  }

  formatConfidence(confidence: number): string {
    return `${Math.round(confidence * 100)}%`;
  }

  formatProvider(provider: string): string {
    const providers: Record<string, string> = {
      openai: 'OpenAI',
      claude: 'Claude',
      gemini: 'Gemini',
      universal: 'Universal'
    };
    return providers[provider] || provider;
  }

  formatBlockType(type: string): string {
    return type.charAt(0).toUpperCase() + type.slice(1).replace(/([A-Z])/g, ' $1');
  }

  formatTimestamp(date: Date): string {
    return date.toLocaleString();
  }

  renderMath(content: string): string {
    // Basic LaTeX to HTML conversion (in a real app, use MathJax or KaTeX)
    return content.replace(/\$\$(.*?)\$\$/g, '<div class="math-display">$1</div>')
                 .replace(/\$(.*?)\$/g, '<span class="math-inline">$1</span>');
  }

  getHighConfidenceBlocks(): ContentBlock[] {
    if (!this.renderResult?.data?.blocks) return [];
    return this.renderResult.data.blocks.filter((block: ContentBlock) => block.confidence > 0.8);
  }

  getRemainingContent(): string {
    // This would filter out already-rendered blocks from the original content
    return this.message.content; // Simplified for now
  }

  getDebugInfo() {
    return {
      strategy: this.currentStrategy,
      hasError: this.hasError,
      isLoading: this.isLoading,
      renderResult: this.renderResult,
      messageMetadata: {
        hasCodeBlocks: this.message.hasCodeBlocks,
        hasMathBlocks: this.message.hasMathBlocks,
        hasDiagrams: this.message.hasDiagrams,
        parsedAt: this.message.parsedAt,
        parsingProvider: this.message.parsingProvider
      }
    };
  }

  // Sanitize markdown input before rendering to mitigate XSS.
  sanitizeContent(input: string | null | undefined): string {
    if (!input) return '';
    return this.messageFormatter.cleanContent(input);
  }
}

// Strategy implementations
class BlocksRenderingStrategy implements ContentRenderingStrategy {
  canHandle(message: EnhancedMessage): boolean {
    if (message.parsedContent) {
      return ParsedContentAnalyzer.shouldUseBlocksRenderer(message.parsedContent);
    }
    return message.hasCodeBlocks || message.hasMathBlocks || false;
  }

  render(message: EnhancedMessage): ContentRenderResult {
    return {
      component: 'blocks',
      data: message.parsedContent || { blocks: [] },
      confidence: message.parsedContent?.confidence || 0.7,
      renderingStrategy: 'blocks_structured'
    };
  }

  getConfidence(): number {
    return 0.9;
  }
}

class HybridRenderingStrategy implements ContentRenderingStrategy {
  canHandle(message: EnhancedMessage): boolean {
    if (!message.parsedContent) return false;

    const hasHighConfidenceBlocks = message.parsedContent.blocks.some(b => b.confidence > 0.8);
    const hasMultipleTypes = new Set(message.parsedContent.blocks.map(b => b.type)).size > 1;

    return hasHighConfidenceBlocks && hasMultipleTypes;
  }

  render(message: EnhancedMessage): ContentRenderResult {
    return {
      component: 'hybrid',
      data: message.parsedContent || { blocks: [] },
      confidence: 0.8,
      renderingStrategy: 'hybrid_blocks_markdown'
    };
  }

  getConfidence(): number {
    return 0.7;
  }
}

class MarkdownRenderingStrategy implements ContentRenderingStrategy {
  canHandle(message: EnhancedMessage): boolean {
    return true; // Always can handle as fallback
  }

  render(message: EnhancedMessage): ContentRenderResult {
    return {
      component: 'markdown',
      data: { content: message.content },
      confidence: 0.6,
      renderingStrategy: 'markdown_default'
    };
  }

  getConfidence(): number {
    return 0.5;
  }
}
