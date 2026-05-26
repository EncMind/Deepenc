import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable, of, catchError } from 'rxjs';
import { map } from 'rxjs/operators';
import {
  ParseRequest,
  ParseResponse,
  ParsedContent,
  EnhancedMessage,
  ContentTypeDetector,
  ParsedContentAnalyzer,
  PROVIDER_CONFIDENCE
} from '../models/content.models';
import { environment } from '../../environments/environment';

@Injectable({
  providedIn: 'root'
})
export class ContentParserService {
  private readonly baseUrl = '/api/parse';
  private debug(...args: unknown[]): void {
    if (!environment.debugMode || !args.length) {
      return;
    }
    console.log('📝 [ContentParser]', ...args);
  }

  constructor(private http: HttpClient) {}

  /**
   * Parse content using the universal parsing endpoint
   */
  parseContent(content: string, provider?: string): Observable<ParsedContent | null> {
    if (!content?.trim()) {
      return of(null);
    }

    const request: ParseRequest = {
      content: content.trim(),
      provider: provider || 'universal'
    };

    const headers = new HttpHeaders({
      'Content-Type': 'application/json'
    });

    return this.http.post<ParseResponse>(`${this.baseUrl}/content`, request, { headers })
      .pipe(
        map(response => this.transformResponse(response)),
        catchError(error => {
          console.error('Content parsing failed:', error);
          return of(null);
        })
      );
  }

  /**
   * Parse content using OpenAI-specific parsing
   */
  parseOpenAI(content: string): Observable<ParsedContent | null> {
    return this.parseWithProvider(content, 'openai', `${this.baseUrl}/openai`);
  }

  /**
   * Parse content using Claude-specific parsing
   */
  parseClaude(content: string): Observable<ParsedContent | null> {
    return this.parseWithProvider(content, 'claude', `${this.baseUrl}/claude`);
  }

  /**
   * Parse content using Gemini-specific parsing
   */
  parseGemini(content: string): Observable<ParsedContent | null> {
    return this.parseWithProvider(content, 'gemini', `${this.baseUrl}/gemini`);
  }

  /**
   * Determine if a message should use parsed content rendering
   */
  shouldUseParsedRendering(message: EnhancedMessage): boolean {
    // If we already have parsed content, analyze it
    if (message.parsedContent) {
      return ParsedContentAnalyzer.shouldUseBlocksRenderer(message.parsedContent);
    }

    // If message has parsing metadata, use it
    if (message.hasCodeBlocks || message.hasMathBlocks || message.hasDiagrams) {
      return true;
    }

    // Fall back to content analysis
    if (message.content) {
      return ContentTypeDetector.needsSpecialProcessing(message.content);
    }

    return false;
  }

  /**
   * Get the appropriate parsing strategy for a provider
   */
  getParsingStrategy(provider: string): 'openai' | 'claude' | 'gemini' | 'universal' {
    switch (provider.toLowerCase()) {
      case 'openai':
      case 'gpt-4':
      case 'gpt-3.5-turbo':
        return 'openai';
      case 'claude':
      case 'anthropic':
        return 'claude';
      case 'gemini':
      case 'google':
        return 'gemini';
      default:
        return 'universal';
    }
  }

  /**
   * Parse content on the client side as a fallback
   */
  parseClientSide(content: string, provider: string = 'universal'): ParsedContent {
    this.debug('Starting client-side parsing');
    this.debug('Client-side content length:', content?.length || 0);
    this.debug('Client-side provider:', provider);

    const blocks = [];
    const confidence = PROVIDER_CONFIDENCE[provider] || 0.7;
    this.debug('Client-side confidence:', confidence);

    // Simple regex-based parsing as fallback
    const hasCodeBlocks = ContentTypeDetector.hasCodeBlocks(content);
    this.debug('Content has code blocks:', hasCodeBlocks);

    if (hasCodeBlocks) {
      const codeRegex = /```(\w*)\n([\s\S]*?)```/g;
      let match;
      let codeBlockCount = 0;
      this.debug('Starting code block extraction');

      while ((match = codeRegex.exec(content)) !== null) {
        codeBlockCount++;
        this.debug(`Found code block ${codeBlockCount}: language=${match[1] || 'text'}, length=${match[2]?.trim()?.length || 0}`);

        blocks.push({
          type: 'code' as const,
          content: match[2].trim(),
          language: match[1] || 'text',
          confidence: 0.8,
          metadata: {
            format: 'client_side_parsing'
          }
        });
      }

      this.debug('Total code blocks found:', codeBlockCount);
    }

    // Add remaining content as text blocks
    let remainingContent = content;
    this.debug('Processing remaining content after code blocks');

    // Remove code blocks
    const originalLength = remainingContent.length;
    remainingContent = remainingContent.replace(/```[\s\S]*?```/g, '');
    this.debug('Content length after removing code blocks:', remainingContent.length, '(removed', originalLength - remainingContent.length, 'chars)');

    // Split into paragraphs
    const paragraphs = remainingContent.split('\n\n').filter(p => p.trim());
    this.debug('Found paragraphs:', paragraphs.length);

    paragraphs.forEach((paragraph, index) => {
      if (paragraph.trim()) {
        this.debug(`Adding text block ${index + 1}: length=${paragraph.trim().length}`);
        blocks.push({
          type: 'text' as const,
          content: paragraph.trim(),
          confidence: 0.6,
          metadata: {
            format: 'client_side_paragraph'
          }
        });
      }
    });

    const result = {
      blocks,
      metadata: {
        strategy: 'client_side_fallback',
        version: '1.0'
      },
      provider: 'client',
      confidence,
      parsedAt: new Date()
    };

    this.debug('Client-side parsing completed');
    this.debug('Final result - total blocks:', result.blocks.length);
    this.debug('Final result - confidence:', result.confidence);
    this.debug('Final result - provider:', result.provider);

    return result;
  }

  /**
   * Enhanced parsing with multiple strategies and fallbacks
   */
  parseWithFallback(content: string, provider: string): Observable<ParsedContent> {
    this.debug('Starting parseWithFallback');
    this.debug('Content length:', content?.length || 0);
    this.debug('Provider:', provider);

    // Do not log content previews to avoid leaking chat content
    // Only log content length for debugging
    this.debug('Content length (no preview):', content?.length || 0);

    const strategy = this.getParsingStrategy(provider);
    this.debug('Selected strategy:', strategy);

    // Try provider-specific parsing first
    return this.parseContent(content, strategy).pipe(
      map(result => {
        this.debug('Parse result received');
        this.debug('Has result:', !!result);
        this.debug('Result confidence:', result?.confidence);
        this.debug('Result blocks count:', result?.blocks?.length || 0);

        if (result && result.confidence > 0.7) {
          this.debug('High confidence result, using it');
          return result;
        }

        // If provider-specific parsing failed or low confidence, try universal
        if (strategy !== 'universal') {
          this.debug(`Provider-specific parsing failed/low confidence, trying universal for ${provider}`);
          // Note: In a real implementation, we might want to make another API call here
          // For now, we'll fall back to client-side parsing
        }

        // Fall back to client-side parsing
        this.debug('Falling back to client-side parsing');
        const clientResult = this.parseClientSide(content, provider);
        this.debug('Client-side parsing completed');
        this.debug('Client-side blocks:', clientResult?.blocks?.length || 0);
        return clientResult;
      }),
      catchError((error) => {
        console.error('📝 [ContentParser] All parsing methods failed:', error);
        this.debug('Using client-side fallback');
        const fallbackResult = this.parseClientSide(content, provider);
        this.debug('Fallback parsing blocks:', fallbackResult?.blocks?.length || 0);
        return of(fallbackResult);
      })
    );
  }

  private parseWithProvider(content: string, provider: string, endpoint: string): Observable<ParsedContent | null> {
    if (!content?.trim()) {
      return of(null);
    }

    const request: ParseRequest = {
      content: content.trim()
    };

    const headers = new HttpHeaders({
      'Content-Type': 'application/json'
    });

    return this.http.post<ParseResponse>(endpoint, request, { headers })
      .pipe(
        map(response => this.transformResponse(response)),
        catchError(error => {
          console.error(`${provider} parsing failed:`, error);
          return of(null);
        })
      );
  }

  private transformResponse(response: ParseResponse): ParsedContent {
    return {
      blocks: response.blocks,
      metadata: response.metadata,
      provider: response.provider,
      confidence: response.confidence,
      parsedAt: new Date(response.parsedAt)
    };
  }
}
