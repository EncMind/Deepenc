// Content parsing models matching the Go backend structures

export interface ContentBlock {
  type: 'text' | 'code' | 'math' | 'list' | 'table' | 'diagram' | 'thinking';
  content: string;
  language?: string;
  metadata?: Record<string, any>;
  confidence: number;
}

export interface ParsedContent {
  blocks: ContentBlock[];
  metadata: Record<string, any>;
  provider: string;
  confidence: number;
  parsedAt: Date;
}

export interface ParseRequest {
  content: string;
  provider?: string;
}

export interface ParseResponse {
  blocks: ContentBlock[];
  metadata: Record<string, any>;
  provider: string;
  confidence: number;
  parsedAt: string; // ISO string from backend
}

// Enhanced message interface that includes parsing information
export interface EnhancedMessage {
  id: string;
  type: string;
  threadId: string;
  userId: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  provider: string;
  model: string;
  createdAt: number;
  vectorId?: string;

  // Encryption fields
  encryptedContent?: string;
  encryptedAESKey?: string;
  iv?: string;
  authTag?: string;
  algorithm?: string;

  // Content parsing fields
  parsedContent?: ParsedContent;
  hasCodeBlocks?: boolean;
  hasMathBlocks?: boolean;
  hasDiagrams?: boolean;
  parsedAt?: number;
  parsingProvider?: string;
}

// Content rendering strategy interface
export interface ContentRenderingStrategy {
  canHandle(message: EnhancedMessage): boolean;
  render(message: EnhancedMessage): ContentRenderResult;
  getConfidence(): number;
}

export interface ContentRenderResult {
  component: 'markdown' | 'blocks' | 'hybrid';
  data: any;
  confidence: number;
  renderingStrategy: string;
}

// Block-specific interfaces
export interface CodeBlock extends ContentBlock {
  type: 'code';
  language: string;
  metadata: {
    format: 'standard_backticks' | 'header_format' | 'claude_backticks' | 'gemini_backticks';
    detected_language?: string;
    pattern_index?: number;
  };
}

export interface MathBlock extends ContentBlock {
  type: 'math';
  metadata: {
    format: 'latex' | 'inline' | 'block';
  };
}

export interface TextBlock extends ContentBlock {
  type: 'text';
  metadata: {
    format: 'paragraph' | 'heading' | 'quote';
  };
}

export interface ListBlock extends ContentBlock {
  type: 'list';
  metadata: {
    list_type: 'ordered' | 'unordered';
  };
}

export interface ThinkingBlock extends ContentBlock {
  type: 'thinking';
  metadata: {
    format: 'claude_thinking';
  };
}

// Provider-specific parsing confidence levels
export const PROVIDER_CONFIDENCE: Record<string, number> = {
  openai: 0.95,
  claude: 0.85,
  gemini: 0.80,
  universal: 0.70
};

// Content type detection utilities
export class ContentTypeDetector {
  static hasCodeBlocks(content: string): boolean {
    return /```[\s\S]*?```|'''[\s\S]*?'''|`[^`\n]+`/.test(content);
  }

  static hasMathNotation(content: string): boolean {
    return /\$\$[\s\S]*?\$\$|\$[^$\n]+\$|\\[\[\(][\s\S]*?\\[\]\)]/.test(content);
  }

  static hasMermaidDiagrams(content: string): boolean {
    return /```mermaid[\s\S]*?```/.test(content);
  }

  static hasThinkingTags(content: string): boolean {
    return /<thinking>[\s\S]*?<\/thinking>/.test(content);
  }

  static needsSpecialProcessing(content: string): boolean {
    return this.hasCodeBlocks(content) ||
           this.hasMathNotation(content) ||
           this.hasMermaidDiagrams(content) ||
           this.hasThinkingTags(content);
  }
}

// Parsing result analysis
export class ParsedContentAnalyzer {
  static getHighestConfidenceBlocks(parsed: ParsedContent): ContentBlock[] {
    return parsed.blocks
      .filter(block => block.confidence > 0.8)
      .sort((a, b) => b.confidence - a.confidence);
  }

  static getCodeBlocks(parsed: ParsedContent): CodeBlock[] {
    return parsed.blocks.filter(block => block.type === 'code') as CodeBlock[];
  }

  static getMathBlocks(parsed: ParsedContent): MathBlock[] {
    return parsed.blocks.filter(block => block.type === 'math') as MathBlock[];
  }

  static getTextBlocks(parsed: ParsedContent): TextBlock[] {
    return parsed.blocks.filter(block => block.type === 'text') as TextBlock[];
  }

  static shouldUseBlocksRenderer(parsed: ParsedContent): boolean {
    const codeBlocks = this.getCodeBlocks(parsed);
    const mathBlocks = this.getMathBlocks(parsed);

    // Use blocks renderer if we have high-confidence code blocks
    if (codeBlocks.length > 0 && codeBlocks.some(block => block.confidence > 0.85)) {
      return true;
    }

    // Use blocks renderer if we have math content
    if (mathBlocks.length > 0) {
      return true;
    }

    // Use blocks renderer if overall confidence is high and we have mixed content
    if (parsed.confidence > 0.85 && parsed.blocks.length > 2) {
      return true;
    }

    return false;
  }
}

// Content formatting utilities
export class ContentFormatter {
  static formatTimestamp(unixTimestamp: number): string {
    return new Date(unixTimestamp * 1000).toLocaleString();
  }

  static formatConfidence(confidence: number): string {
    return `${Math.round(confidence * 100)}%`;
  }

  static formatProvider(provider: string): string {
    switch (provider) {
      case 'openai': return 'OpenAI';
      case 'claude': return 'Claude';
      case 'gemini': return 'Gemini';
      case 'universal': return 'Universal';
      default: return provider;
    }
  }

  static formatBlockType(type: string): string {
    switch (type) {
      case 'code': return 'Code Block';
      case 'math': return 'Math Expression';
      case 'text': return 'Text';
      case 'list': return 'List';
      case 'thinking': return 'Thinking';
      default: return type;
    }
  }
}