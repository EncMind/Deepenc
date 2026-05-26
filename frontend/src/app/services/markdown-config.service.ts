import { Injectable } from '@angular/core';
import { MarkedOptions } from 'ngx-markdown';

@Injectable({
  providedIn: 'root'
})
export class MarkdownConfigService {
  private codeBlockObserver: MutationObserver | null = null;
  private observerTarget: Element | null = null;

  getMarkedOptions(): MarkedOptions {
    // BetterChatGPT approach: minimal config, let marked handle GFM
    return {
      breaks: true  // Enable line breaks like BetterChatGPT
    };
  }

  // Get ngx-markdown options with Prism.js configuration
  getOptions() {
    return {
      markedOptions: this.getMarkedOptions(),
      // Enable sanitization to mitigate XSS from rendered markdown
      sanitize: true,
      katex: true,     // Enable KaTeX for math
      mermaid: false,  // Disable mermaid for now to avoid conflicts
      prismPlugin: true, // Enable Prism.js highlighting
    };
  }

  // Safe preprocessing to handle user content before markdown parsing
  preprocessContent(content: string): string {
    if (!content) return '';

    // Step 1: Decode HTML entities
    let processed = this.decodeHtmlEntities(content);

    // Step 2: Fix code block boundary issues
    processed = this.fixCodeBlockBoundaries(processed);

    return processed;
  }

  // Fix code block boundary detection issues
  private fixCodeBlockBoundaries(content: string): string {
    let fixed = content;

    // Ensure code blocks have proper spacing
    // Add blank lines before and after code blocks if missing
    fixed = fixed.replace(/([^\n])\n```/g, '$1\n\n```');
    fixed = fixed.replace(/```\n([^\n])/g, '```\n\n$1');

    // Fix cases where code blocks might be split across paragraphs
    // Look for incomplete code blocks and try to repair them
    fixed = this.repairSplitCodeBlocks(fixed);

    return fixed;
  }

  // Attempt to repair code blocks that may have been split
  private repairSplitCodeBlocks(content: string): string {
    const lines = content.split('\n');
    const result: string[] = [];
    let inCodeBlock = false;
    let codeBlockStart = -1;

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];

      // Check for code block start
      if (line.startsWith('```') && !inCodeBlock) {
        inCodeBlock = true;
        codeBlockStart = i;
        result.push(line);
      }
      // Check for code block end
      else if (line.startsWith('```') && inCodeBlock) {
        inCodeBlock = false;
        result.push(line);
      }
      // Regular content
      else {
        result.push(line);
      }
    }

    // If we ended in a code block without closing, try to add closing backticks
    if (inCodeBlock && codeBlockStart >= 0) {
      result.push('```');
    }

    return result.join('\n');
  }

  // Apply additional content fixes for common issues
  private applyContentFixes(content: string): string {
    let fixed = content;

    // Fix spacing around code blocks
    fixed = fixed.replace(/([^\n])(```)/g, '$1\n\n$2');
    fixed = fixed.replace(/(```[^\n]*\n)([\s\S]*?)(```)/g, (match, opening, codeContent, closing) => {
      // Clean up the code content
      let cleanedCode = codeContent;

      // Remove excessive indentation
      const lines = cleanedCode.split('\n');
      if (lines.length > 1) {
        const nonEmptyLines = lines.filter(line => line.trim().length > 0);
        if (nonEmptyLines.length > 0) {
          const minIndent = Math.min(...nonEmptyLines.map(line => {
            const match = line.match(/^(\s*)/);
            return match ? match[1].length : 0;
          }));

          if (minIndent > 0) {
            cleanedCode = lines.map(line => {
              if (line.trim().length === 0) return line;
              return line.substring(minIndent);
            }).join('\n');
          }
        }
      }

      return opening + cleanedCode + closing;
    });

    // Add newline after code blocks
    fixed = fixed.replace(/(```\n?)([^\n])/g, '$1\n$2');

    return fixed;
  }

  // Decode HTML entities but preserve markdown structure
  private decodeHtmlEntities(content: string): string {
    // Create a temporary element to decode HTML entities
    if (typeof document !== 'undefined') {
      const textarea = document.createElement('textarea');
      textarea.innerHTML = content;
      return textarea.value;
    }

    // Fallback for server-side rendering
    return content
      .replace(/&lt;/g, '<')
      .replace(/&gt;/g, '>')
      .replace(/&amp;/g, '&')
      .replace(/&quot;/g, '"')
      .replace(/&#39;/g, "'")
      .replace(/&nbsp;/g, ' ');
  }

  // Normalize different code block formats to standard markdown
  private normalizeCodeBlocks(content: string): string {
    let normalized = content;

    // Handle code blocks that start and end with triple quotes
    normalized = this.convertTripleQuotesToBackticks(normalized);

    // Ensure proper code block language detection
    normalized = normalized.replace(/```(\w+)\s*\n/g, '```$1\n');

    // Fix common indentation issues in code blocks
    normalized = this.fixCodeBlockIndentation(normalized);

    // Add language hints for common patterns
    normalized = this.addLanguageHints(normalized);

    return normalized;
  }

  // Convert various triple quote patterns to markdown code blocks
  private convertTripleQuotesToBackticks(content: string): string {
    let converted = content;

    // Handle the specific pattern from your example: '''...(code)...'''
    // Match triple quotes at start and end of lines with code content between
    converted = converted.replace(/^'''\s*$([\s\S]*?)^'''\s*$/gm, '```\n$1\n```');

    // Handle inline triple quotes
    converted = converted.replace(/'''([\s\S]*?)'''/g, (match, codeContent) => {
      // If content spans multiple lines, use block format
      if (codeContent.includes('\n')) {
        return '```\n' + codeContent.trim() + '\n```';
      } else {
        // Single line code
        return '`' + codeContent.trim() + '`';
      }
    });

    // Handle triple double quotes similarly
    converted = converted.replace(/"""([\s\S]*?)"""/g, (match, codeContent) => {
      if (codeContent.includes('\n')) {
        return '```\n' + codeContent.trim() + '\n```';
      } else {
        return '`' + codeContent.trim() + '`';
      }
    });

    // Handle mixed patterns
    converted = converted.replace(/'''([\s\S]*?)```/g, '```\n$1\n```');
    converted = converted.replace(/```([\s\S]*?)'''/g, '```\n$1\n```');

    // Final cleanup - any remaining triple quotes become triple backticks
    converted = converted.replace(/'''/g, '```');
    converted = converted.replace(/"""/g, '```');

    return converted;
  }

  // Add language hints based on content patterns
  private addLanguageHints(content: string): string {
    return content.replace(/```\n([\s\S]*?)\n```/g, (match, code) => {
      const trimmedCode = code.trim();

      // Python detection
      if (/import\s+\w+|def\s+\w+\s*\(|class\s+\w+|from\s+\w+\s+import/.test(trimmedCode)) {
        return '```python\n' + code + '\n```';
      }

      // JavaScript/TypeScript detection
      if (/function\s+\w+|const\s+\w+\s*=|let\s+\w+\s*=|var\s+\w+\s*=|import.*from|export/.test(trimmedCode)) {
        return '```javascript\n' + code + '\n```';
      }

      // SQL detection
      if (/SELECT\s+|INSERT\s+|UPDATE\s+|DELETE\s+|CREATE\s+|ALTER\s+|DROP\s+/i.test(trimmedCode)) {
        return '```sql\n' + code + '\n```';
      }

      // JSON detection
      if (/^\s*[\{\[][\s\S]*[\}\]]\s*$/.test(trimmedCode)) {
        return '```json\n' + code + '\n```';
      }

      // Return as-is if no language detected
      return match;
    });
  }

  // Fix indentation issues within code blocks
  private fixCodeBlockIndentation(content: string): string {
    return content.replace(/```(\w*)\n([\s\S]*?)\n```/g, (match, lang, code) => {
      // Split the code into lines
      const lines = code.split('\n');

      // Normalize tabs to spaces (1 tab = 2 spaces for consistency)
      const normalizedLines = lines.map(line => line.replace(/\t/g, '  '));

      // Find non-empty lines, excluding leading and trailing empty lines
      let startIndex = 0;
      let endIndex = normalizedLines.length - 1;

      // Skip empty lines at the start
      while (startIndex < normalizedLines.length && normalizedLines[startIndex].trim().length === 0) {
        startIndex++;
      }

      // Skip empty lines at the end
      while (endIndex >= 0 && normalizedLines[endIndex].trim().length === 0) {
        endIndex--;
      }

      if (startIndex > endIndex) return match; // All lines are empty

      // Get the content lines (excluding leading/trailing empty lines)
      const contentLines = normalizedLines.slice(startIndex, endIndex + 1);
      const nonEmptyContentLines = contentLines.filter(line => line.trim().length > 0);

      if (nonEmptyContentLines.length === 0) return match;

      // Find the minimum indentation from non-empty content lines
      const minIndent = Math.min(...nonEmptyContentLines.map(line => {
        const match = line.match(/^(\s*)/);
        return match ? match[1].length : 0;
      }));

      // Remove the minimum indentation from all lines, preserving empty lines
      const dedentedLines = normalizedLines.map((line, index) => {
        // Keep leading and trailing empty lines as-is
        if (index < startIndex || index > endIndex) {
          return line;
        }
        // For content lines, remove minimum indentation
        if (line.trim().length === 0) {
          return ''; // Clean up empty lines within content
        }
        return line.substring(minIndent);
      });

      return '```' + lang + '\n' + dedentedLines.join('\n') + '\n```';
    });
  }

  // Initialize syntax highlighting with secure settings
  initializePrismJS(): void {
    if (typeof window !== 'undefined' && (window as any).Prism) {
      const Prism = (window as any).Prism;

      // Disable autoloader since we're loading components explicitly in angular.json
      if (Prism.plugins && Prism.plugins.autoloader) {
        // Disable autoloader to prevent 404 errors
        Prism.plugins.autoloader = null;
      }

      // Security: Disable HTML parsing in code blocks
      Prism.hooks.add('wrap', (env: any) => {
        if (env.type === 'string' || env.type === 'comment') {
          env.content = env.content.replace(/</g, '&lt;').replace(/>/g, '&gt;');
        }
      });

      // Manual highlight to ensure components are loaded
      setTimeout(() => {
        Prism.highlightAll();
      }, 100);
    }
  }

  // Initialize KaTeX with secure settings
  initializeKaTeX(): void {
    if (typeof window !== 'undefined' && (window as any).katex) {
      const katex = (window as any).katex;

      // Configure auto-render with security settings
      if (katex.render) {
        // Security: Strict mode prevents dangerous commands
        const defaultOptions = {
          strict: true,
          throwOnError: false,
          trust: false,
          maxSize: 10,
          maxExpand: 100
        };

        // Store default options for later use
        (window as any).katexDefaultOptions = defaultOptions;
      }
    }
  }

  // Initialize Mermaid with secure settings
  initializeMermaid(): void {
    if (typeof window !== 'undefined' && (window as any).mermaid) {
      const mermaid = (window as any).mermaid;

      mermaid.initialize({
        startOnLoad: false,
        theme: 'neutral',
        securityLevel: 'strict',
        deterministicIds: true,
        maxTextSize: 50000,
        maxEdges: 500,
        fontFamily: 'inherit'
      });
    }
  }

  // Add copy buttons to code blocks
  addCopyButtonsToCodeBlocks(): void {
    if (typeof document === 'undefined') {
      return;
    }
    const preElements = document.querySelectorAll('pre:not(.has-copy-btn)');
    preElements.forEach(pre => this.attachCopyButton(pre));
  }

  private attachCopyButton(pre: Element): void {
    if (!pre || pre.classList.contains('has-copy-btn')) {
      return;
    }
    const codeElement = pre.querySelector('code');
    if (!codeElement) {
      return;
    }

    pre.classList.add('has-copy-btn');

    const copyButton = document.createElement('button');
    copyButton.className = 'code-copy-btn';
    copyButton.innerHTML = '📋 Copy';
    copyButton.title = 'Copy code';
    copyButton.type = 'button';
    copyButton.style.cssText = `
      position: absolute !important;
      top: 8px !important;
      right: 8px !important;
      background: #f1f5f9 !important;
      color: #1f2937 !important;
      border: 1px solid #cbd5e1 !important;
      padding: 3px 8px !important;
      border-radius: 5px !important;
      font-size: 0.7rem !important;
      z-index: 100 !important;
      opacity: 1 !important;
      visibility: visible !important;
      transition: none !important;
      animation: none !important;
      transform: none !important;
    `;

    pre.appendChild(copyButton);
  }

  // Set up mutation observer to handle dynamically added content
  private setupCodeBlockObserver(target: Element): void {
    if (typeof MutationObserver === 'undefined') {
      return;
    }

    if (this.codeBlockObserver) {
      if (this.observerTarget === target) {
        return; // already observing this element
      }
      this.codeBlockObserver.disconnect();
      this.codeBlockObserver = null;
      this.observerTarget = null;
    }

    const observer = new MutationObserver(mutations => {
      mutations.forEach(mutation => {
        if (mutation.type !== 'childList' || mutation.addedNodes.length === 0) {
          return;
        }
        mutation.addedNodes.forEach(node => this.processNodeForCopyButtons(node));
      });
    });

    observer.observe(target, {
      childList: true,
      subtree: true
    });

    this.codeBlockObserver = observer;
    this.observerTarget = target;
    (window as any).codeBlockObserver = observer;
  }

  private processNodeForCopyButtons(node: Node): void {
    if (node.nodeType !== Node.ELEMENT_NODE) {
      return;
    }

    const element = node as Element;
    if (element.tagName === 'PRE') {
      this.attachCopyButton(element);
      return;
    }

    if (element.querySelector) {
      const nestedPres = element.querySelectorAll('pre:not(.has-copy-btn)');
      nestedPres.forEach(pre => this.attachCopyButton(pre));
    }
  }

  // Initialize all components
  initializeAll(): void {
    if (typeof window !== 'undefined') {
      // Wait for scripts to load
      setTimeout(() => {
        this.initializePrismJS();
        this.initializeKaTeX();
        this.initializeMermaid();

        // Add copy buttons to existing code blocks
        this.addCopyButtonsToCodeBlocks();

        const target = document.querySelector('.chat-interface');
        if (target) {
          this.setupCodeBlockObserver(target);
        }
      }, 100);

      // Also add copy buttons after a longer delay to catch any late-rendered content
      setTimeout(() => {
        this.addCopyButtonsToCodeBlocks();
        const target = document.querySelector('.chat-interface');
        if (target) {
          this.setupCodeBlockObserver(target);
        }
      }, 500);
    }
  }
}
