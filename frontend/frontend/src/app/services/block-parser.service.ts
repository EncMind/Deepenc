import { Injectable } from '@angular/core';

export interface TextBlock {
  type: 'markdown' | 'code';
  content: string;
  language?: string;
  isPartial?: boolean;
}

@Injectable({
  providedIn: 'root'
})
export class BlockParserService {

  parseTextIntoBlocks(text: string): TextBlock[] {
    if (!text) return [];

    // First try standard triple backticks
    const standardMatches = this.findStandardCodeBlocks(text);

    if (standardMatches.length > 0) {
      return this.buildBlocksFromMatches(text, standardMatches);
    }

    // Fallback: Look for "Code:" followed by content
    const codeHeaderMatches = this.findCodeHeaderBlocks(text);

    if (codeHeaderMatches.length > 0) {
      return this.buildBlocksFromMatches(text, codeHeaderMatches);
    }

    // Last resort: treat entire text as markdown
    return [{
      type: 'markdown',
      content: text
    }];
  }

  private findStandardCodeBlocks(text: string): any[] {
    const standardRegex = /```([^\n`]*)\n([\s\S]*?)(```(?=[ *\n]|$)|$)/g;
    const matches = [];
    let match;

    while ((match = standardRegex.exec(text)) !== null) {
      matches.push({
        start: match.index,
        end: match.index + match[0].length,
        language: (match[1] || '').trim(),
        code: match[2],
        isPartial: !match[3].startsWith('```')
      });
    }

    return matches;
  }

  private findCodeHeaderBlocks(text: string): any[] {
    // Look for "Code:" header followed by content
    const codeHeaderRegex = /^Code:\s*$/gm;
    const matches = [];
    let match;

    while ((match = codeHeaderRegex.exec(text)) !== null) {
      const headerEnd = match.index + match[0].length;
      const remainingText = text.substring(headerEnd);

      // Try different patterns to capture code after "Code:"
      const patterns = [
        // Pattern 1: Everything until next major section
        /^\s*\n([\s\S]*?)(?=\n\n[A-Z][^:]*:|$)/m,
        // Pattern 2: Everything until "Notes:" or similar
        /^\s*\n([\s\S]*?)(?=\n\nNotes:|$)/m,
        // Pattern 3: Simple everything after newline
        /^\s*\n([\s\S]*?)$/m
      ];

      for (const pattern of patterns) {
        const codeMatch = remainingText.match(pattern);
        if (codeMatch && codeMatch[1].trim()) {
          const codeContent = this.cleanCodeBlock(codeMatch[1]);
          if (codeContent.trim()) {
            matches.push({
              start: headerEnd + codeMatch.index + codeMatch[0].indexOf(codeMatch[1]),
              end: headerEnd + codeMatch.index + codeMatch[0].length,
              language: this.detectLanguage(codeContent),
              code: codeContent,
              isPartial: false
            });
            break; // Use first successful pattern
          }
        }
      }
    }

    return matches;
  }

  private cleanCodeBlock(rawCode: string): string {
    // Clean up the code block, removing excessive whitespace and handling indentation
    const lines = rawCode.split('\n');

    // Remove leading and trailing empty lines
    while (lines.length > 0 && lines[0].trim() === '') {
      lines.shift();
    }
    while (lines.length > 0 && lines[lines.length - 1].trim() === '') {
      lines.pop();
    }

    if (lines.length === 0) return '';

    // Find minimum indentation of non-empty lines
    const nonEmptyLines = lines.filter(line => line.trim().length > 0);
    if (nonEmptyLines.length === 0) return rawCode;

    const minIndent = Math.min(...nonEmptyLines.map(line => {
      const match = line.match(/^(\s*)/);
      return match ? match[1].length : 0;
    }));

    // Remove minimum indentation from all lines
    const cleanLines = lines.map(line => {
      if (line.trim().length === 0) return '';
      return line.substring(Math.min(minIndent, line.length - line.trimStart().length));
    });

    return cleanLines.join('\n');
  }

  private buildBlocksFromMatches(text: string, matches: any[]): TextBlock[] {
    const blocks: TextBlock[] = [];
    let lastIndex = 0;

    // Sort matches by position
    matches.sort((a, b) => a.start - b.start);

    for (const match of matches) {
      // Add any text before the code block as markdown
      if (match.start > lastIndex) {
        const markdownContent = text.slice(lastIndex, match.start).trim();
        if (markdownContent) {
          blocks.push({
            type: 'markdown',
            content: markdownContent
          });
        }
      }

      // Add the code block
      blocks.push({
        type: 'code',
        content: match.code,
        language: match.language || 'plaintext',
        isPartial: match.isPartial
      });

      lastIndex = match.end;
    }

    // Add any remaining text as markdown
    if (lastIndex < text.length) {
      const remainingContent = text.slice(lastIndex).trim();
      if (remainingContent) {
        blocks.push({
          type: 'markdown',
          content: remainingContent
        });
      }
    }

    return blocks;
  }

  // Helper method to count lines for display purposes
  countLines(text: string): number {
    return text.split('\n').length;
  }

  // Detect language from code content if not specified
  detectLanguage(code: string): string {
    const trimmedCode = code.trim();

    // Python detection
    if (/import\s+\w+|def\s+\w+\s*\(|class\s+\w+|from\s+\w+\s+import/.test(trimmedCode)) {
      return 'python';
    }

    // JavaScript/TypeScript detection
    if (/function\s+\w+|const\s+\w+\s*=|let\s+\w+\s*=|var\s+\w+\s*=|import.*from|export/.test(trimmedCode)) {
      return 'javascript';
    }

    // SQL detection
    if (/SELECT\s+|INSERT\s+|UPDATE\s+|DELETE\s+|CREATE\s+|ALTER\s+|DROP\s+/i.test(trimmedCode)) {
      return 'sql';
    }

    // JSON detection
    if (/^\s*[\{\[][\s\S]*[\}\]]\s*$/.test(trimmedCode)) {
      return 'json';
    }

    // HTML detection
    if (/<!DOCTYPE html|<html|<\/html>/.test(trimmedCode)) {
      return 'html';
    }

    // Shell/Bash detection
    if (/^#!\/bin\/bash|^\s*\$\s+|cd\s+|ls\s+|grep\s+/.test(trimmedCode)) {
      return 'bash';
    }

    return 'plaintext';
  }
}