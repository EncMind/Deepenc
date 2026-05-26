import { Injectable } from '@angular/core';
import { MarkdownConfigService } from './markdown-config.service';

export interface FormattedMessage {
  content: string;
  hasCodeBlocks: boolean;
  hasMath: boolean;
  hasDiagrams: boolean;
  isProcessed: boolean;
}

@Injectable({
  providedIn: 'root'
})
export class MessageFormatterService {

  constructor(private markdownConfig: MarkdownConfigService) {}

  // Main method to process message content for display
  formatMessage(content: string): FormattedMessage {
    if (!content) {
      return {
        content: '',
        hasCodeBlocks: false,
        hasMath: false,
        hasDiagrams: false,
        isProcessed: true
      };
    }

    // Apply minimal preprocessing and sanitize to mitigate XSS
    let processedContent = this.minimalPreprocessing(content);
    processedContent = this.cleanContent(processedContent);

    // Analyze content features (for optimization/loading decisions)
    const hasCodeBlocks = this.hasCodeBlocks(content);
    const hasMath = this.hasMathNotation(content);
    const hasDiagrams = this.hasMermaidDiagrams(content);

    return {
      content: processedContent,
      hasCodeBlocks,
      hasMath,
      hasDiagrams,
      isProcessed: true
    };
  }

  // Minimal preprocessing following BetterChatGPT's pattern
  private minimalPreprocessing(content: string): string {
    if (!content) return '';

    // EXTREMELY minimal - just decode HTML entities and that's it
    let processed = content;

    if (content.includes('&gt;') || content.includes('&lt;') || content.includes('&amp;')) {
      // Only decode HTML entities, no other processing
      processed = this.decodeHtmlEntitiesOnly(processed);
    }

    return processed;
  }

  // Simple HTML entity decoding without other processing
  private decodeHtmlEntitiesOnly(content: string): string {
    if (typeof document !== 'undefined') {
      const textarea = document.createElement('textarea');
      textarea.innerHTML = content;
      return textarea.value;
    }

    // Fallback manual decoding
    return content
      .replace(/&lt;/g, '<')
      .replace(/&gt;/g, '>')
      .replace(/&amp;/g, '&')
      .replace(/&quot;/g, '"')
      .replace(/&#39;/g, "'")
      .replace(/&nbsp;/g, ' ');
  }

  // Detect if content contains code blocks
  private hasCodeBlocks(content: string): boolean {
    return /```[\s\S]*?```|'''[\s\S]*?'''|`[^`\n]+`/.test(content);
  }

  // Detect if content contains math notation
  private hasMathNotation(content: string): boolean {
    return /\$\$[\s\S]*?\$\$|\$[^$\n]+\$|\\[\[\(][\s\S]*?\\[\]\)]/.test(content);
  }

  // Detect if content contains Mermaid diagrams
  private hasMermaidDiagrams(content: string): boolean {
    return /```mermaid[\s\S]*?```/.test(content);
  }

  // Apply formatting fixes to improve code display
  private applyFormattingFixes(content: string): string {
    let fixed = content;

    // Fix non-standard code block markers
    fixed = this.fixCodeBlockMarkers(fixed);

    // Fix common code block issues
    fixed = this.fixCodeBlockFormatting(fixed);

    // Fix inline code formatting
    fixed = this.fixInlineCodeFormatting(fixed);

    // Fix list formatting
    fixed = this.fixListFormatting(fixed);

    // Fix table formatting
    fixed = this.fixTableFormatting(fixed);

    return fixed;
  }

  // Fix non-standard code block markers
  private fixCodeBlockMarkers(content: string): string {
    // Convert triple single quotes to triple backticks
    let fixed = content.replace(/'''/g, '```');

    // Convert triple double quotes to triple backticks
    fixed = fixed.replace(/"""/g, '```');

    // Fix mixed quote patterns
    fixed = fixed.replace(/```'/g, '```');
    fixed = fixed.replace(/'```/g, '```');
    fixed = fixed.replace(/```"/g, '```');
    fixed = fixed.replace(/"```/g, '```');

    return fixed;
  }

  // Fix code block formatting issues
  private fixCodeBlockFormatting(content: string): string {
    // Ensure proper spacing around code blocks
    let fixed = content.replace(/([^\n])(\n```)/g, '$1\n\n```');
    fixed = fixed.replace(/(```\n?)([^\n])/g, '$1\n$2');
    fixed = fixed.replace(/([^\n])(```)/g, '$1\n\n```');

    // Fix language specification spacing
    fixed = fixed.replace(/```\s+(\w+)/g, '```$1');

    // Ensure code blocks end with newlines
    fixed = fixed.replace(/(```[\s\S]*?)```([^\n])/g, '$1```\n\n$2');

    return fixed;
  }

  // Fix inline code formatting
  private fixInlineCodeFormatting(content: string): string {
    // Fix spacing around inline code
    let fixed = content.replace(/([^\s])`([^`]+)`([^\s])/g, '$1 `$2` $3');

    // Fix inline code in middle of sentences
    fixed = fixed.replace(/(\w)`(\w[^`]*\w)`(\w)/g, '$1 `$2` $3');

    return fixed;
  }

  // Fix list formatting
  private fixListFormatting(content: string): string {
    let fixed = content;

    // Ensure proper spacing before lists
    fixed = fixed.replace(/([^\n])(\n[-*+]\s)/g, '$1\n\n$2');
    fixed = fixed.replace(/([^\n])(\n\d+\.\s)/g, '$1\n\n$2');

    // Fix nested list indentation
    fixed = fixed.replace(/^(\s*)([-*+])\s+/gm, '$1$2 ');
    fixed = fixed.replace(/^(\s*)(\d+\.)\s+/gm, '$1$2 ');

    return fixed;
  }

  // Fix table formatting
  private fixTableFormatting(content: string): string {
    let fixed = content;

    // Ensure proper spacing around tables
    fixed = fixed.replace(/([^\n])(\n\|.*\|)/g, '$1\n\n$2');
    fixed = fixed.replace(/(\|.*\|)([^\n])/g, '$1\n\n$2');

    // Fix table header separators
    fixed = fixed.replace(/(\|.*\|)\n(\|[-\s:]*\|)/g, '$1\n$2');

    return fixed;
  }

  // Escape dangerous content patterns
  private escapeHtml(content: string): string {
    return content
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  // Clean up content for safe display
  cleanContent(content: string): string {
    if (!content) return '';

    // Remove potentially dangerous patterns
    let cleaned = content
      .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
      .replace(/javascript:/gi, '')
      .replace(/data:text\/html/gi, '');

    // Remove inline event handlers (onclick, onload, etc.) only within actual HTML tags
    cleaned = cleaned.replace(/<[^>]*>/gi, (tag) => {
      return tag.replace(/\s+on[\w-]+\s*=\s*(?:"[^"]*"|'[^']*')/gi, '');
    });

    // Normalize whitespace
    cleaned = cleaned.replace(/\r\n/g, '\n').replace(/\r/g, '\n');

    // Remove excessive blank lines
    cleaned = cleaned.replace(/\n{3,}/g, '\n\n');

    return cleaned.trim();
  }

  // Check if content needs special processing
  needsSpecialProcessing(content: string): boolean {
    return this.hasCodeBlocks(content) ||
           this.hasMathNotation(content) ||
           this.hasMermaidDiagrams(content);
  }
}
