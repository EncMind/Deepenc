import {
  Component,
  OnInit,
  OnDestroy,
  ViewEncapsulation,
  ViewChild,
  signal,
  NgZone
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { MarkdownModule } from 'ngx-markdown';
import { ChatHistoryComponent } from './chat-history/chat-history.component';
import { ChatService, StreamWithContextRequest } from './chat.service';
import { AuthService } from './auth.service';
import { MessageFormatterService } from './services/message-formatter.service';
import { MarkdownConfigService } from './services/markdown-config.service';
import { MessageSendLock, ChatPanel } from './shared/message-send-lock';
import { environment } from '../environments/environment';
import 'highlight.js/styles/github.css';
import hljs from 'highlight.js';

interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
  timestamp?: string;
  formattedContent?: string;
  isStreaming?: boolean;
}

interface ChatThread {
  threadId: string | null;
  messages: ChatMessage[];
}

interface Thread {
  id: string;
  type: string;
  userId: string;
  threadId: string;
  title: string;
  createdAt: number;
  updatedAt: number;
  metadata: {
    messageCount: number;
    lastModel: string;
  };
  status: string;
}

interface ModelsResponse {
  openai: string[];
  anthropic: string[];
  gemini: string[];
  defaultOpenAI: string;
  defaultAnthropic: string;
  defaultGemini: string;
}

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, FormsModule, ChatHistoryComponent, MarkdownModule],
  encapsulation: ViewEncapsulation.None,
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent implements OnInit, OnDestroy {
  @ViewChild(ChatHistoryComponent) chatHistoryComponent?: ChatHistoryComponent;
  
  status = signal<string>('Loading...');
  models = signal<ModelsResponse | null>(null);

  leftModel = signal<string>('');
  centerModel = signal<string>('');
  rightModel = signal<string>('');
  userInput = signal<string>('');
  sendLock = new MessageSendLock();

  // Thread management - each chat maintains its own thread
  leftThread = signal<ChatThread>({ threadId: null, messages: [] });
  centerThread = signal<ChatThread>({ threadId: null, messages: [] });
  rightThread = signal<ChatThread>({ threadId: null, messages: [] });

  // For display compatibility with existing template
  leftMessages = signal<ChatMessage[]>([]);
  centerMessages = signal<ChatMessage[]>([]);
  rightMessages = signal<ChatMessage[]>([]);

  leftChatOpen = signal<boolean>(true);
  centerChatOpen = signal<boolean>(true);
  rightChatOpen = signal<boolean>(true);

  // Track streaming state to avoid expensive markdown parsing during streaming
  leftStreaming = signal<boolean>(false);
  centerStreaming = signal<boolean>(false);
  rightStreaming = signal<boolean>(false);

  // History sidebar
  showHistory = signal<boolean>(false);
  selectedThread = signal<Thread | null>(null);
  isLoadingThread = signal<boolean>(false);

  private readonly maxScrollAttempts = 3;
  private scrollPendingChats = new Map<'left' | 'center' | 'right', number>();
  private scrollFrameId: number | null = null;
  private scrollFrameCancel: (() => void) | null = null;
  private highlightPendingChats = new Set<'left' | 'center' | 'right'>();
  private highlightFrameId: number | null = null;
  private highlightFrameCancel: (() => void) | null = null;

  // Global update batching for all chats
  private pendingUpdates = new Map<'left' | 'center' | 'right', string[]>();
  private updateTimerId: number | null = null;

  // Scroll throttling - only scroll every 500ms during streaming
  private lastScrollTime = new Map<'left' | 'center' | 'right', number>();

  // Context configuration - token-aware is now the default
  private contextConfig = {
    ragEnabled: true,
    strategy: 'hybrid' as 'recent' | 'rag' | 'hybrid'
    // Note: token-aware processing is now enabled by default when model is provided
  };

  // Markdown configuration
  markdownOptions: any;

  constructor(
    private api: ChatService,
    private zone: NgZone,
    private authService: AuthService,
    private route: ActivatedRoute,
    private messageFormatter: MessageFormatterService,
    private markdownConfig: MarkdownConfigService
  ) {}

  ngOnInit() {
    // Initialize markdown configuration
    this.markdownOptions = this.markdownConfig.getMarkedOptions();
    this.markdownConfig.initializeAll();

    // Show UI immediately, load data in background
    this.status.set('Loading...');

    // Load data non-blocking
    this.loadModelsAsync();
    this.loadChosenModelsAsync();
    
    this.initializeThreads();
    this.setupNewChatListener();

    // No longer using query parameters for history state

    // Expose copy helper (optional)
    (window as any).copyToClipboard = this.copyToClipboard.bind(this);

    // Listen in CAPTURE phase to reliably catch clicks on nested elements
    document.addEventListener('click', this.handleCopyButtonClick, true);
  }

  ngOnDestroy(): void {
    document.removeEventListener('click', this.handleCopyButtonClick, true);
    window.removeEventListener('newChat', this.handleNewChatEvent);
    window.removeEventListener('toggleHistory', this.handleToggleHistoryEvent);
    if (this.scrollFrameCancel) {
      this.scrollFrameCancel();
      this.scrollFrameCancel = null;
    } else if (typeof window !== 'undefined' && this.scrollFrameId !== null && typeof window.cancelAnimationFrame === 'function') {
      window.cancelAnimationFrame(this.scrollFrameId);
    }
    this.scrollFrameId = null;
    this.scrollPendingChats.clear();
    if (this.highlightFrameCancel) {
      this.highlightFrameCancel();
      this.highlightFrameCancel = null;
    } else if (typeof window !== 'undefined' && this.highlightFrameId !== null && typeof window.cancelAnimationFrame === 'function') {
      window.cancelAnimationFrame(this.highlightFrameId);
    }
    this.highlightFrameId = null;
    this.highlightPendingChats.clear();

    if (this.updateTimerId !== null) {
      window.clearTimeout(this.updateTimerId);
      this.updateTimerId = null;
    }
    this.pendingUpdates.clear();
    this.sendLock.reset();
  }

  // ============================
  // Thread Management
  // ============================

  async initializeThreads() {
    // Wait for authentication to complete before initializing threads
    const maxWaitTime = 10000; // 10 seconds
    const startTime = Date.now();
    
    while (!this.api.isAuthenticated() && (Date.now() - startTime) < maxWaitTime) {
      await new Promise(resolve => setTimeout(resolve, 500)); // Wait 500ms
    }
    
    if (!this.api.isAuthenticated()) {
      console.error('Authentication timeout - cannot create threads');
      return;
    }

    // Don't create threads on startup - they will be created when user sends first message
    this.debugLog('Thread system initialized - threads will be created when needed');
  }

  async ensureThreadsExist() {
    try {
      // Double check authentication before proceeding
      if (!this.api.isAuthenticated()) {
        console.error('User not authenticated, cannot create threads');
        return;
      }
      
      // Threads will be created automatically when first message is sent
      // This allows automatic title generation based on user's first message
      this.debugLog('🔧 Threads will be created automatically when messages are sent for auto-generated titles');
    } catch (error) {
      console.error('Failed to create threads:', error);
    }
  }

  async createNewThreads() {
    // Legacy method - now just calls ensureThreadsExist
    await this.ensureThreadsExist();
  }

  // ============================
  // History Management
  // ============================

  onThreadSelected(thread: Thread) {
    this.debugLog(`Thread selected: ${thread.title}`);
    this.selectedThread.set(thread);
    this.loadThreadConversation(thread);
  }

  onThreadDeleted(threadId: string) {
    this.debugLog(`Thread deleted: ${threadId}`);
    // If the deleted thread was active, clear the selected thread
    if (this.selectedThread()?.id === threadId) {
      this.selectedThread.set(null);
    }
  }

  onHistoryClosed() {
    this.debugLog('Chat history closed by user');
    this.showHistory.set(false);
  }

  async loadThreadConversation(thread: Thread) {
    this.isLoadingThread.set(true);
    try {
      this.debugLog(`Loading conversation for thread: ${thread.id}`);
      
      // Get the messages for this thread
      const response = await this.api.getThreadMessages(thread.id);
      const messages = response.messages;
      
      this.debugLog(`Loaded ${messages.length} messages for thread ${thread.id}`);
      
      // Clear current conversations and load the selected thread's messages
      this.clearAllChats();
      
      // Load messages into appropriate chat based on the last model used
      if (thread.metadata.lastModel.includes('gpt') || thread.metadata.lastModel.includes('openai')) {
        this.loadMessagesIntoChat('left', messages, thread.id);
      } else if (thread.metadata.lastModel.includes('claude') || thread.metadata.lastModel.includes('anthropic')) {
        this.loadMessagesIntoChat('center', messages, thread.id);
      } else if (thread.metadata.lastModel.includes('gemini') || thread.metadata.lastModel.includes('google')) {
        this.loadMessagesIntoChat('right', messages, thread.id);
      } else {
        // Default to left chat if model is unknown
        this.loadMessagesIntoChat('left', messages, thread.id);
      }
      
    } catch (error) {
      console.error('Error loading thread conversation:', error);
    } finally {
      this.isLoadingThread.set(false);
    }
  }

  private loadMessagesIntoChat(chat: 'left' | 'center' | 'right', messages: any[], threadId: string) {
    // Convert enhanced messages to chat messages
    const chatMessages: ChatMessage[] = messages.map(msg => {
      const base: ChatMessage = {
        role: msg.role,
        content: msg.content || '',
        timestamp: new Date(msg.createdAt).toLocaleTimeString(),
        isStreaming: false
      };

      if (msg.role === 'assistant') {
        base.formattedContent = this.messageFormatter.formatMessage(base.content).content;
      }

      return base;
    });

    // Update the appropriate chat
    switch (chat) {
      case 'left':
        this.leftThread.update(t => ({ ...t, threadId, messages: chatMessages }));
        this.leftMessages.set(chatMessages);
        break;
      case 'center':
        this.centerThread.update(t => ({ ...t, threadId, messages: chatMessages }));
        this.centerMessages.set(chatMessages);
        break;
      case 'right':
        this.rightThread.update(t => ({ ...t, threadId, messages: chatMessages }));
        this.rightMessages.set(chatMessages);
        break;
    }
  }

  private clearAllChats() {
    this.leftMessages.set([]);
    this.centerMessages.set([]);
    this.rightMessages.set([]);
    this.leftThread.update(t => ({ ...t, messages: [] }));
    this.centerThread.update(t => ({ ...t, messages: [] }));
    this.rightThread.update(t => ({ ...t, messages: [] }));
  }

  toggleHistory() {
    this.showHistory.update(show => {
      const newShowValue = !show;
      // If we're showing the history, refresh the threads
      if (newShowValue && this.chatHistoryComponent) {
        // Use setTimeout to ensure the component is fully rendered
        setTimeout(() => {
          this.debugLog('Refreshing chat history on toggle...');
          this.chatHistoryComponent?.loadThreads();
        }, 100);
      }
      return newShowValue;
    });
  }

  // ============================
  // Models & UI helpers
  // ============================

  private async loadModelsAsync() {
    try {
      const models = await this.api.getModels();
      this.models.set(models);
    } catch (error) {
      console.error('Failed to load models:', error);
      this.status.set('Error loading models');
    }
  }

  private resolveModelChoice(preferred: string | undefined, options: string[], fallback?: string): string {
    if (preferred && options.includes(preferred)) {
      return preferred;
    }
    if (options.length > 0) {
      return options[0];
    }
    if (fallback) {
      return fallback;
    }
    return preferred || '';
  }

  private async loadChosenModelsAsync() {
    try {
      const [chosenModels, models] = await Promise.all([
        this.api.getChosenModels(),
        this.api.getModels().catch(() => null)
      ]);

      // Ensure the models signal is populated if we fetched them here
      if (models && !this.models()) {
        this.models.set(models);
      }

      const availableModels = this.models();

      const openaiOptions = availableModels?.openai || [];
      const anthropicOptions = availableModels?.anthropic || [];
      const geminiOptions = availableModels?.gemini || [];

      this.leftModel.set(
        this.resolveModelChoice(
          chosenModels.openai,
          openaiOptions,
          availableModels?.defaultOpenAI
        )
      );
      this.centerModel.set(
        this.resolveModelChoice(
          chosenModels.anthropic,
          anthropicOptions,
          availableModels?.defaultAnthropic
        )
      );
      this.rightModel.set(
        this.resolveModelChoice(
          chosenModels.gemini,
          geminiOptions,
          availableModels?.defaultGemini
        )
      );
      this.status.set(''); // Clear loading status
    } catch (error) {
      console.error('Failed to load chosen models:', error);
      // Use defaults if API fails
      this.status.set('');
    }
  }

  openaiModels(): string[]     { return this.models() ? (this.models()!.openai || [])     : []; }
  anthropicModels(): string[]  { return this.models() ? (this.models()!.anthropic || [])  : []; }
  geminiModels(): string[]     { return this.models() ? (this.models()!.gemini || [])     : []; }

  getDefaultChatGPTModel(): string   { return this.models()?.defaultOpenAI || (this.openaiModels().length ? this.openaiModels()[0] : 'No models available'); }
  getDefaultClaudeAIModel(): string  { return this.models()?.defaultAnthropic || (this.anthropicModels().length ? this.anthropicModels()[0] : 'No models available'); }
  getDefaultGeminiModel(): string    { return this.models()?.defaultGemini || (this.geminiModels().length ? this.geminiModels()[0] : 'No models available'); }

  getModelDisplayName(modelId: string): string {
    if (!modelId) return '';
    return this.api.getModelDisplayName(modelId);
  }

  areModelsLoaded(): boolean {
    return this.models() !== null &&
      (this.openaiModels().length > 0 || this.anthropicModels().length > 0 || this.geminiModels().length > 0);
  }

  onLeftModelChange(event: Event): void   { 
    const newModel = (event.target as HTMLSelectElement).value;
    this.leftModel.set(newModel);
    // No permanent save - just change for this session
  }
  onCenterModelChange(event: Event): void { 
    const newModel = (event.target as HTMLSelectElement).value;
    this.centerModel.set(newModel);
    // No permanent save - just change for this session
  }
  onRightModelChange(event: Event): void  { 
    const newModel = (event.target as HTMLSelectElement).value;
    this.rightModel.set(newModel);
    // No permanent save - just change for this session
  }
  onUserInputChange(event: Event): void   { this.userInput.set((event.target as HTMLTextAreaElement).value); }

  onKeyDown(event: KeyboardEvent): void {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      if (this.sendLock.isLocked()) {
        this.status.set('Please wait for the current response to finish.');
        return;
      }
      this.sendMessage();
    }
  }

  getTimestamp(): string { return new Date().toLocaleTimeString(); }

  toggleLeftChat(): void   { this.leftChatOpen.update(open => !open); }
  toggleCenterChat(): void { this.centerChatOpen.update(open => !open); }
  toggleRightChat(): void  { this.rightChatOpen.update(open => !open); }
  expandAllChats(): void   { this.leftChatOpen.set(true); this.centerChatOpen.set(true); this.rightChatOpen.set(true); }

  // ============================
  // Clipboard (capture-phase delegation)
  // ============================

  copyMessage(content: string, event?: Event) {
    const button = event?.target as HTMLElement;
    const original = button?.textContent || '📋 Copy';

    const okUI = () => {
      if (button) {
        button.textContent = '✅ Copied!';
        (button as HTMLElement).style.background = '#48bb78';
        setTimeout(() => { 
          button.textContent = original; 
          (button as HTMLElement).style.background = '#f1f5f9'; 
        }, 1800);
      }
    };
    const failUI = () => {
      if (button) {
        button.textContent = '❌ Failed';
        (button as HTMLElement).style.background = '#e53e3e';
        setTimeout(() => { 
          button.textContent = original; 
          (button as HTMLElement).style.background = '#f1f5f9'; 
        }, 1800);
      }
    };

    const tryAsync = async () => {
      if (navigator.clipboard && (window as any).isSecureContext) {
        await navigator.clipboard.writeText(content);
        return true;
      }
      return false;
    };

    const tryTextarea = () => {
      const ta = document.createElement('textarea');
      ta.value = content; ta.readOnly = true;
      ta.style.position = 'fixed'; ta.style.top = '0'; ta.style.left = '-9999px';
      document.body.appendChild(ta);
      ta.select(); ta.setSelectionRange(0, ta.value.length);
      let ok = false; try { ok = document.execCommand('copy'); } catch { ok = false; }
      document.body.removeChild(ta);
      return ok;
    };

    tryAsync().then(ok => ok ? okUI() : (tryTextarea() ? okUI() : failUI())).catch(() => tryTextarea() ? okUI() : failUI());
  }

  copyToClipboard(button: HTMLElement) {
    const block = button.closest('.code-block') as HTMLElement | null;
    const codeEl = block?.querySelector('.code-content') as HTMLElement | null;
    if (!codeEl) return;

    const text = codeEl.innerText || codeEl.textContent || '';
    const original = button.textContent || '📋 Copy';

    const okUI = () => {
      button.textContent = '✓ Copied';
      button.classList.add('copied');
      setTimeout(() => { 
        button.textContent = original; 
        button.classList.remove('copied');
      }, 2000);
    };
    const failUI = () => {
      button.textContent = '✗ Failed';
      button.style.color = '#ef4444';
      button.style.background = 'rgba(239, 68, 68, 0.1)';
      setTimeout(() => { 
        button.textContent = original; 
        button.style.color = '';
        button.style.background = '';
      }, 2000);
    };

    const tryAsync = async () => {
      if (navigator.clipboard && (window as any).isSecureContext) {
        await navigator.clipboard.writeText(text);
        return true;
      }
      return false;
    };

    const tryTextarea = () => {
      const ta = document.createElement('textarea');
      ta.value = text; ta.readOnly = true;
      ta.style.position = 'fixed'; ta.style.top = '0'; ta.style.left = '-9999px';
      document.body.appendChild(ta);
      ta.select(); ta.setSelectionRange(0, ta.value.length);
      let ok = false; try { ok = document.execCommand('copy'); } catch { ok = false; }
      document.body.removeChild(ta);
      return ok;
    };

    const tryRange = () => {
      const range = document.createRange(); range.selectNodeContents(codeEl);
      const sel = window.getSelection(); sel?.removeAllRanges(); sel?.addRange(range);
      let ok = false; try { ok = document.execCommand('copy'); } catch { ok = false; }
      sel?.removeAllRanges(); return ok;
    };

    (async () => {
      try { if (await tryAsync()) { okUI(); return; } } catch { /* ignore */ }
      (tryTextarea() || tryRange()) ? okUI() : failUI();
    })();
  }

  // Copy code from code blocks
  copyCodeToClipboard(button: HTMLElement) {
    // Find the pre element that contains this button
    const preElement = button.closest('pre') as HTMLElement | null;
    if (!preElement) return;

    // Find the code element within the pre
    const codeElement = preElement.querySelector('code') as HTMLElement | null;
    if (!codeElement) return;

    // Get the text content
    const text = codeElement.textContent || codeElement.innerText || '';
    const original = button.textContent || '📋 Copy';

    const showSuccess = () => {
      button.textContent = '✅ Copied!';
      setTimeout(() => {
        button.textContent = original;
      }, 2000);
    };

    const showFail = () => {
      button.textContent = '❌ Failed';
      setTimeout(() => {
        button.textContent = original;
      }, 2000);
    };

    const tryAsync = async (): Promise<boolean> => {
      if (navigator.clipboard && (window as any).isSecureContext) {
        try {
          await navigator.clipboard.writeText(text);
          return true;
        } catch {
          return false;
        }
      }
      return false;
    };

    const tryTextarea = (): boolean => {
      const ta = document.createElement('textarea');
      ta.value = text;
      ta.readOnly = true;
      ta.style.position = 'fixed';
      ta.style.top = '0';
      ta.style.left = '-9999px';
      document.body.appendChild(ta);
      ta.select();
      ta.setSelectionRange(0, ta.value.length);
      let ok = false;
      try {
        ok = document.execCommand('copy');
      } catch {
        ok = false;
      }
      document.body.removeChild(ta);
      return ok;
    };

    // Try to copy the text
    tryAsync().then(ok => ok ? showSuccess() : (tryTextarea() ? showSuccess() : showFail()))
             .catch(() => tryTextarea() ? showSuccess() : showFail());
  }

  private handleCopyButtonClick = (event: Event): void => {
    const path = (event as any).composedPath?.() as EventTarget[] | undefined;
    let btn: HTMLElement | null = null;

    if (Array.isArray(path)) {
      for (const n of path) {
        if (n instanceof Element && (n.classList.contains('copy-button') || n.classList.contains('copy-btn') || n.classList.contains('code-copy-btn'))) {
          btn = n as HTMLElement; break;
        }
      }
    }
    if (!btn && event.target) {
      const start = event.target instanceof Element ? event.target : (event.target as any).parentElement;
      if (start) btn = (start as Element).closest('.copy-button, .copy-btn, .code-copy-btn') as HTMLElement | null;
    }
    if (btn) {
      // Handle code copy buttons differently than regular copy buttons
      if (btn.classList.contains('code-copy-btn')) {
        this.copyCodeToClipboard(btn);
      } else {
        this.copyToClipboard(btn);
      }
    }
  };


  // ============================
  // Rendering helpers (Markdown‑lite + code blocks)
  // ============================

  private esc(s: string): string {
    return s.replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  private renderInline(raw: string): string {
    // First, restore any pre-processed content (for Gemini)
    let processed = raw
      .replace(/__SUP_START__([^_]+?)__SUP_END__/g, '<sup>$1</sup>')
      .replace(/__SUB_START__([^_]+?)__SUB_END__/g, '<sub>$1</sub>');

    // Handle math variables BEFORE extracting code blocks
    // This ensures math vars are converted first
    processed = processed.replace(/__MATH_VAR__(.*?)__MATH_VAR_END__/g, (match, inner) => {
      // Don't escape the content inside math vars as it's already safe
      return `<span class="math-var">${inner}</span>`;
    });

    // Extract inline code (but not the math vars we just converted)
    const codes: string[] = [];
    processed = processed.replace(/`([^`]+)`/g, (_m, code) => {
      codes.push(code);
      return `__IC_${codes.length - 1}__`;
    });

    // Handle LaTeX/math notation - fix nested issues
    processed = processed
      // Handle \( ... \) for inline math
      .replace(/\\\((.*?)\\\)/g, (_m, math) => {
        const cleaned = this.cleanMathExpression(math);
        return `__MATH_INLINE_START__${cleaned}__MATH_INLINE_END__`;
      })
      // Handle \[ ... \] for display math
      .replace(/\\\[(.*?)\\\]/g, (_m, math) => {
        const cleaned = this.cleanMathExpression(math);
        return `__MATH_DISPLAY_START__${cleaned}__MATH_DISPLAY_END__`;
      })
      // Handle $...$ notation
      .replace(/\$([^$]+)\$/g, (_m, math) => {
        const cleaned = this.cleanMathExpression(math);
        return `__MATH_INLINE_START__${cleaned}__MATH_INLINE_END__`;
      });

    // Escape HTML (but preserve our math-var spans)
    let out = processed
      .split('<span class="math-var">')
      .map((part, index) => {
        if (index === 0) {
          return this.esc(part);
        }
        const [mathContent, ...rest] = part.split('</span>');
        return `<span class="math-var">${mathContent}</span>` + this.esc(rest.join('</span>'));
      })
      .join('');

    // **bold**
    out = out.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
    // *italic* (avoid conflict with **)
    out = out.replace(/(^|[^*])\*(?!\s)([^*]+?)\*(?!\*)/g, (_m, pre, body) => `${pre}<em>${body}</em>`);

    // Restore inline code placeholders
    out = out.replace(/__IC_(\d+)__/g, (_m, i) => {
      const code = this.esc(codes[parseInt(i, 10)] || '');
      return `<code class="inline-code">${code}</code>`;
    });

    // Restore math expressions (after escaping to prevent nested HTML issues)
    out = out.replace(/__MATH_INLINE_START__(.*?)__MATH_INLINE_END__/g,
      '<span class="math-inline">$1</span>');
    out = out.replace(/__MATH_DISPLAY_START__(.*?)__MATH_DISPLAY_END__/g,
      '<div class="math-display">$1</div>');

    // Also restore any that were escaped
    out = out.replace(/&lt;sup&gt;/g, '<sup>')
      .replace(/&lt;\/sup&gt;/g, '</sup>')
      .replace(/&lt;sub&gt;/g, '<sub>')
      .replace(/&lt;\/sub&gt;/g, '</sub>');

    return out;
  }

  private cleanMathExpression(math: string): string {
    // Clean up common LaTeX expressions to readable format
    let cleaned = math
      .trim()
      // Spacing commands (handle before other replacements)
      .replace(/\\quad/g, '    ')
      .replace(/\\qquad/g, '        ')
      .replace(/\\,/g, ' ')
      .replace(/\\:/g, ' ')
      .replace(/\\;/g, ' ')
      .replace(/\\!/g, '')
      .replace(/\\ /g, ' ')
      // Basic symbols
      .replace(/\\cdot/g, '·')
      .replace(/\\times/g, '×')
      .replace(/\\div/g, '÷')
      .replace(/\\pm/g, '±')
      .replace(/\\mp/g, '∓')
      .replace(/\\neq/g, '≠')
      .replace(/\\approx/g, '≈')
      .replace(/\\equiv/g, '≡')
      .replace(/\\leq/g, '≤')
      .replace(/\\geq/g, '≥')
      .replace(/\\ll/g, '≪')
      .replace(/\\gg/g, '≫')
      // Binary operators
      .replace(/\\oplus/g, '⊕')
      .replace(/\\ominus/g, '⊖')
      .replace(/\\otimes/g, '⊗')
      .replace(/\\oslash/g, '⊘')
      .replace(/\\odot/g, '⊙')
      .replace(/\\circ/g, '∘')
      .replace(/\\bullet/g, '•')
      .replace(/\\star/g, '⋆')
      .replace(/\\ast/g, '*')
      // Greek letters
      .replace(/\\alpha/g, 'α')
      .replace(/\\beta/g, 'β')
      .replace(/\\gamma/g, 'γ')
      .replace(/\\delta/g, 'δ')
      .replace(/\\epsilon/g, 'ε')
      .replace(/\\theta/g, 'θ')
      .replace(/\\lambda/g, 'λ')
      .replace(/\\mu/g, 'μ')
      .replace(/\\pi/g, 'π')
      .replace(/\\sigma/g, 'σ')
      .replace(/\\tau/g, 'τ')
      .replace(/\\phi/g, 'φ')
      .replace(/\\omega/g, 'ω')
      // Arrows
      .replace(/\\rightarrow/g, '→')
      .replace(/\\leftarrow/g, '←')
      .replace(/\\Rightarrow/g, '⇒')
      .replace(/\\Leftarrow/g, '⇐')
      .replace(/\\leftrightarrow/g, '↔')
      // Sets
      .replace(/\\in/g, '∈')
      .replace(/\\notin/g, '∉')
      .replace(/\\subset/g, '⊂')
      .replace(/\\subseteq/g, '⊆')
      .replace(/\\cup/g, '∪')
      .replace(/\\cap/g, '∩')
      .replace(/\\emptyset/g, '∅')
      // Logic
      .replace(/\\forall/g, '∀')
      .replace(/\\exists/g, '∃')
      .replace(/\\neg/g, '¬')
      .replace(/\\land/g, '∧')
      .replace(/\\lor/g, '∨')
      .replace(/\\wedge/g, '∧')
      .replace(/\\vee/g, '∨')
      // Other common symbols
      .replace(/\\infty/g, '∞')
      .replace(/\\partial/g, '∂')
      .replace(/\\nabla/g, '∇')
      .replace(/\\sum/g, 'Σ')
      .replace(/\\prod/g, 'Π')
      .replace(/\\int/g, '∫')
      // Common functions
      .replace(/\\sin/g, 'sin')
      .replace(/\\cos/g, 'cos')
      .replace(/\\tan/g, 'tan')
      .replace(/\\log/g, 'log')
      .replace(/\\ln/g, 'ln')
      .replace(/\\exp/g, 'exp')
      .replace(/\\mod/g, 'mod')
      .replace(/\\bmod/g, 'mod')
      // Handle text{...} commands
      .replace(/\\text\{([^}]+)\}/g, '$1')
      .replace(/\\mathrm\{([^}]+)\}/g, '$1')
      .replace(/\\mathit\{([^}]+)\}/g, '$1')
      .replace(/\\mathbf\{([^}]+)\}/g, '$1')
      .replace(/\\operatorname\{([^}]+)\}/g, '$1')
      // Fractions (simple handling)
      .replace(/\\frac\{([^}]+)\}\{([^}]+)\}/g, '($1/$2)')
      // Superscripts and subscripts
      .replace(/\^(\w)/g, '<sup>$1</sup>')
      .replace(/\^{([^}]+)}/g, '<sup>$1</sup>')
      .replace(/_(\w)/g, '<sub>$1</sub>')
      .replace(/_{([^}]+)}/g, '<sub>$1</sub>')
      // Square root
      .replace(/\\sqrt\{([^}]+)\}/g, '√($1)')
      .replace(/\\sqrt(\w)/g, '√$1')
      // Remove remaining backslashes for simple variables
      .replace(/\\([a-zA-Z]+)/g, '$1')
      // Clean up extra spaces
      .replace(/\s+/g, ' ');

    // Escape any HTML that might interfere with our math rendering
    cleaned = cleaned
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      // But preserve our intentional sup/sub tags
      .replace(/&lt;sup&gt;/g, '<sup>')
      .replace(/&lt;\/sup&gt;/g, '</sup>')
      .replace(/&lt;sub&gt;/g, '<sub>')
      .replace(/&lt;\/sub&gt;/g, '</sub>');

    return cleaned;
  }

  private renderMarkdownish(text: string): string {
    const src = (text || '').replace(/\r\n/g, '\n');

    // 1) Pull out fenced code blocks first (preserve content verbatim)
    const codeBlocks: string[] = [];
    let working = src.replace(/```(\w+)?\n([\s\S]*?)```/g, (_m, lang, code) => {
      const language = (lang || 'text') as string;
      const escaped = this.esc(code as string);
      const html =
        '<div class="code-block">' +
        '<div class="code-header">' +
        '<span class="language-label">' + (language.toUpperCase()) + '</span>' +
        '<span class="copy-button" role="button" tabindex="0">📋 Copy</span>' +
        '</div>' +
        '<pre class="code-content"><code class="hljs language-' + language + '">' + escaped + '</code></pre>' +
        '</div>';
      codeBlocks.push(html);
      return '__CODE_BLOCK_' + (codeBlocks.length - 1) + '__';
    });


    const lines = working.split('\n');
    const html: string[] = [];
    let para: string[] = [];
    let inUL = false, inOL = false, inTable = false;

    const closePara = () => {
      if (para.length) {
        html.push(`<p>${this.renderInline(para.join(' '))}</p>`);
        para = [];
      }
    };
    const closeLists = () => {
      if (inUL) { html.push('</ul>'); inUL = false; }
      if (inOL) { html.push('</ol>'); inOL = false; }
    };
    const closeTable = () => {
      if (inTable) { html.push('</tbody></table>'); inTable = false; }
    };

    for (const rawLine of lines) {
      const line = rawLine ?? '';

      // Placeholder for fenced code blocks
      if (/^__CODE_BLOCK_\d+__$/.test(line.trim())) {
        closePara(); closeLists();
        html.push(line.trim());
        continue;
      }

      // Blank line -> end paragraph/list/table
      if (/^\s*$/.test(line)) {
        closePara(); closeLists(); closeTable();
        continue;
      }

      // Headings
      const h = line.match(/^(#{1,6})\s+(.*)$/);
      if (h) {
        closePara(); closeLists();
        const level = Math.min(6, h[1].length);
        html.push(`<h${level}>${this.renderInline(h[2])}</h${level}>`);
        continue;
      }

      // Blockquote
      const bq = line.match(/^\s*>\s+(.*)$/);
      if (bq) {
        closePara(); closeLists();
        html.push(`<blockquote>${this.renderInline(bq[1])}</blockquote>`);
        continue;
      }

      // Unordered list
      const ul = line.match(/^\s*[-*]\s+(.*)$/);
      if (ul) {
        closePara();
        if (!inUL) { closeLists(); html.push('<ul>'); inUL = true; }
        html.push(`<li>${this.renderInline(ul[1])}</li>`);
        continue;
      }

      // Ordered list
      const ol = line.match(/^\s*\d+\.\s+(.*)$/);
      if (ol) {
        closePara();
        if (!inOL) { closeLists(); closeTable(); html.push('<ol>'); inOL = true; }
        html.push(`<li>${this.renderInline(ol[1])}</li>`);
        continue;
      }

      // Table detection (pipe-separated values)
      if (line.includes('|')) {
        const cells = line.split('|').map(cell => cell.trim()).filter(cell => cell !== '');
        
        // Check if this is a separator line (like | --- | --- | --- |)
        const isSeparator = cells.every(cell => /^[-:\s]+$/.test(cell));
        
        if (!isSeparator && cells.length > 0) {
          closePara(); closeLists();
          
          // If not in table, start a new table and treat first row as header
          if (!inTable) {
            html.push('<table class="markdown-table"><thead><tr>');
            cells.forEach(cell => {
              html.push(`<th>${this.renderInline(cell)}</th>`);
            });
            html.push('</tr></thead><tbody>');
            inTable = true;
          } else {
            // Regular table row
            html.push('<tr>');
            cells.forEach(cell => {
              html.push(`<td>${this.renderInline(cell)}</td>`);
            });
            html.push('</tr>');
          }
          continue;
        } else if (isSeparator) {
          // Skip separator lines, but keep table state
          continue;
        }
      }

      // If we were in a table but this line doesn't contain pipes, close the table
      if (inTable && !line.includes('|')) {
        closeTable();
      }

      // Otherwise, paragraph buffer
      para.push(line.trim());
    }

    closePara(); closeLists(); closeTable();

    // Restore fenced code blocks
    let rendered = html.join('\n');
    rendered = rendered.replace(/__CODE_BLOCK_(\d+)__/g, (_m, i) => codeBlocks[parseInt(i, 10)]);

    return rendered;
  }

  /**
   * Debug helper for development
   */
  private debugLog(message: string, data?: any): void {
    if (environment.debugMode) {
      console.log(`[DeepEnc Debug] ${message}`, data);
    }
  }

  /**
   * Check if content looks like code based on patterns and structure
   */
  private isLikelyCodeContent(content: string): boolean {
    const lines = content.split('\n');
    const nonEmptyLines = lines.filter(l => l.trim().length > 0);
    
    // If too few lines, not code
    if (nonEmptyLines.length < 3) return false;
    
    let codeIndicators = 0;
    let totalLines = nonEmptyLines.length;
    
    for (const line of lines) {
      const trimmed = line.trim();
      
      // Python/JavaScript/Java indicators
      if (trimmed.startsWith('import ') || trimmed.startsWith('from ') ||
          trimmed.startsWith('def ') || trimmed.startsWith('class ') ||
          trimmed.startsWith('function ') || trimmed.startsWith('const ') ||
          trimmed.startsWith('let ') || trimmed.startsWith('var ') ||
          trimmed.startsWith('if ') || trimmed.startsWith('for ') ||
          trimmed.startsWith('while ') || trimmed.startsWith('try ') ||
          trimmed.startsWith('catch ') || trimmed.startsWith('return ') ||
          trimmed.startsWith('print(') || trimmed.startsWith('console.') ||
          trimmed.startsWith('# ') || trimmed.startsWith('// ') ||
          trimmed.startsWith('/*') || trimmed.endsWith('*/') ||
          trimmed.startsWith('/**') || trimmed.startsWith(' * ')) {
        codeIndicators++;
      }
      
      // Assignment operators
      else if (trimmed.includes(' = ') && !trimmed.includes('==') && !trimmed.includes('!=')) {
        codeIndicators++;
      }
      
      // Method calls or function calls
      else if (/\w+\([^)]*\)/.test(trimmed)) {
        codeIndicators++;
      }
      
      // Indented lines (likely code blocks)
      else if (line.startsWith('    ') || line.startsWith('\t')) {
        codeIndicators++;
      }
      
      // Common programming symbols
      else if (trimmed.includes('->') || trimmed.includes('=>') ||
               trimmed.includes('::') || trimmed.includes('&&') ||
               trimmed.includes('||') || trimmed.includes('++') ||
               trimmed.includes('--') || trimmed.includes('+=')) {
        codeIndicators++;
      }
    }
    
    // If more than 60% of lines look like code, treat as code
    return (codeIndicators / Math.max(totalLines, 1)) > 0.6;
  }

  /**
   * Process mixed content that contains both text and code sections
   */
  private processMixedContent(content: string): string {
    const lines = content.split('\n');
    let result = '';
    let currentCodeBlock = '';
    let currentTextBlock = '';
    let inCodeSection = false;
    
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      const trimmed = line.trim();
      
      // Strong code indicators that suggest start of a code section
      const isCodeLine = this.isStrongCodeLine(trimmed);
      
      // If we find a clear code line and we're not in code section, start one
      if (isCodeLine && !inCodeSection) {
        // Flush any accumulated text
        if (currentTextBlock.trim()) {
          result += this.renderMarkdownish(currentTextBlock.trim()) || `<p>${this.esc(currentTextBlock.trim())}</p>`;
          currentTextBlock = '';
        }
        inCodeSection = true;
        currentCodeBlock = line;
      }
      // If we're in a code section
      else if (inCodeSection) {
        // Check if this looks like end of code (empty line followed by text, or clear text line)
        const nextLine = i + 1 < lines.length ? lines[i + 1] : '';
        const isEndOfCode = (
          (trimmed === '' && nextLine.trim() && !this.isStrongCodeLine(nextLine.trim())) ||
          (trimmed && !this.isStrongCodeLine(trimmed) && !this.couldBeCodeLine(trimmed))
        );
        
        if (isEndOfCode && trimmed === '') {
          // Empty line - end the code block
          if (currentCodeBlock.trim()) {
            result += this.createCodeBlock(currentCodeBlock.trim());
            currentCodeBlock = '';
          }
          inCodeSection = false;
        } else if (isEndOfCode) {
          // Text line - end code block and start text
          if (currentCodeBlock.trim()) {
            result += this.createCodeBlock(currentCodeBlock.trim());
            currentCodeBlock = '';
          }
          inCodeSection = false;
          currentTextBlock = line;
        } else {
          // Continue code block
          currentCodeBlock += '\n' + line;
        }
      }
      // We're in text section
      else {
        currentTextBlock += (currentTextBlock ? '\n' : '') + line;
      }
    }
    
    // Flush remaining content
    if (currentCodeBlock.trim()) {
      result += this.createCodeBlock(currentCodeBlock.trim());
    }
    if (currentTextBlock.trim()) {
      result += this.renderMarkdownish(currentTextBlock.trim()) || `<p>${this.esc(currentTextBlock.trim())}</p>`;
    }
    
    // If we didn't process anything, return original
    return result || content;
  }

  /**
   * Check if a line is a strong indicator of code
   */
  private isStrongCodeLine(line: string): boolean {
    return !!(
      line.match(/^def\s+\w+\s*\(/) ||           // Python function definition
      line.match(/^class\s+\w+/) ||              // Class definition  
      line.match(/^import\s+\w+/) ||             // Import statement
      line.match(/^from\s+\w+\s+import/) ||      // From import
      line.match(/^\s*\w+\s*=\s*\w+\s*\(/) ||   // Function call assignment
      line.match(/^for\s+\w+\s+in\s+/) ||       // For loop
      line.match(/^if\s+.*:$/) ||                // If statement ending with :
      line.match(/^return\s+\w+/) ||             // Return statement
      line.match(/^\s{4,}\w/) ||                 // Indented code (4+ spaces)
      line.match(/^\t+\w/)                       // Tab-indented code
    );
  }

  /**
   * Check if a line could plausibly be part of a code block
   */
  private couldBeCodeLine(line: string): boolean {
    return !!(
      line.includes(' = ') ||
      (line.includes('(') && line.includes(')')) ||
      (line.includes('[') && line.includes(']')) ||
      (line.includes('{') && line.includes('}')) ||
      line.startsWith('  ') ||  // Some indentation
      line.includes('::') ||
      line.includes('->') ||
      line.includes('=>') ||
      line.match(/^\s*\w+:/) ||  // Dictionary-like syntax
      line.match(/^\s*#/)        // Comment
    );
  }

  /**
   * Create a formatted code block
   */
  private createCodeBlock(code: string): string {
    const escaped = this.esc(code);
    return '<div class="code-block">' +
      '<div class="code-header">' +
      '<span class="language-label">PYTHON</span>' +
      '<span class="copy-button" role="button" tabindex="0">📋 Copy</span>' +
      '</div>' +
      '<pre class="code-content" style="white-space: pre-wrap;"><code>' + escaped + '</code></pre>' +
      '</div>';
  }

  /**
   * LibreChat-style: Simple streaming detection based only on incomplete markdown blocks
   */
  private isStreamingContent(content: string): boolean {
    // Only check for incomplete markdown code blocks (LibreChat approach)
    return this.hasIncompleteCodeBlock(content);
  }

  /**
   * Check if content has an incomplete code block (opened but not closed)
   */
  hasIncompleteCodeBlock(content: string): boolean {
    const openBlocks = (content.match(/```/g) || []).length;
    return openBlocks % 2 !== 0; // Odd number means unclosed block
  }

  /**
   * Format streaming content with minimal processing to avoid formatting mess
   */
  formatRawStreamingContent(content: string): string {
    // Escape HTML and preserve whitespace, but don't process markdown
    const escaped = this.esc(content);
    return `<pre class="streaming-content">${escaped}</pre>`;
  }

  /**
   * LibreChat-style content classification
   */
  private classifyContent(content: string): 'streaming' | 'markdown' | 'heuristic-code' | 'text' {
    // 1. Check for incomplete markdown (streaming)
    if (this.hasIncompleteCodeBlock(content)) {
      return 'streaming';
    }
    
    // 2. Check for complete markdown blocks
    if (/```/.test(content)) {
      return 'markdown';
    }
    
    // 3. Fallback to heuristic code detection
    if (this.isLikelyCodeContent(content)) {
      return 'heuristic-code';
    }
    
    // 4. Default to text
    return 'text';
  }


  /**
   * Final formatter used by template: returns safe HTML string (Angular sanitizes by default).
   */
  getFormattedMessage(content: string): string {
    if (!content) return '';
    const formatted = this.messageFormatter.formatMessage(content);
    return formatted.content;
  }


  // ============================
  // Sending & streaming with context
  // ============================

  private collectActiveChats(): ChatPanel[] {
    const chats: ChatPanel[] = [];
    if (this.leftChatOpen() && this.leftModel()) {
      chats.push('left');
    }
    if (this.centerChatOpen() && this.centerModel()) {
      chats.push('center');
    }
    if (this.rightChatOpen() && this.rightModel()) {
      chats.push('right');
    }
    return chats;
  }

  private finalizeSendLock(chat: ChatPanel): void {
    this.sendLock.release(chat);
    if (!this.sendLock.isLocked()) {
      const currentStatus = this.status();
      if (
        currentStatus === 'Generating response...' ||
        currentStatus === 'Please wait for the current response to finish.'
      ) {
        this.status.set('');
      }
    }
  }

  async sendMessage() {
    if (this.sendLock.isLocked()) {
      this.status.set('Please wait for the current response to finish.');
      return;
    }

    const trimmed = this.userInput().trim();
    if (!trimmed) return;

    const activeChats = this.collectActiveChats();
    if (activeChats.length === 0) {
      this.status.set('Open at least one chat to start a conversation.');
      return;
    }

    const msg = trimmed;
    this.userInput.set('');

    const ts = this.getTimestamp();

    // Add user message to all open chats
    this.leftMessages.update(m => [...m, { role: 'user', content: msg, timestamp: ts }]);
    this.centerMessages.update(m => [...m, { role: 'user', content: msg, timestamp: ts }]);
    this.rightMessages.update(m => [...m, { role: 'user', content: msg, timestamp: ts }]);

    // Update thread messages
    this.leftThread.update(t => ({ ...t, messages: [...t.messages, { role: 'user', content: msg, timestamp: ts }] }));
    this.centerThread.update(t => ({ ...t, messages: [...t.messages, { role: 'user', content: msg, timestamp: ts }] }));
    this.rightThread.update(t => ({ ...t, messages: [...t.messages, { role: 'user', content: msg, timestamp: ts }] }));

    this.scrollToBottom('left');
    this.scrollToBottom('center');
    this.scrollToBottom('right');

    this.sendLock.begin(activeChats);
    this.status.set('Generating response...');

    try {
      // Threads will be created automatically by backend with auto-generated titles
      
      // Left (OpenAI)
      if (this.leftModel() && this.leftChatOpen()) {
        this.streamToChat('left', 'openai', this.leftModel(), msg, this.leftThread().threadId);
      }

      // Center (Anthropic) - small delay to prevent simultaneous requests
      if (this.centerModel() && this.centerChatOpen()) {
        setTimeout(() => {
          this.streamToChat('center', 'anthropic', this.centerModel(), msg, this.centerThread().threadId);
        }, 100);
      }

      // Right (Gemini) - small delay to prevent simultaneous requests
      if (this.rightModel() && this.rightChatOpen()) {
        setTimeout(() => {
          this.streamToChat('right', 'gemini', this.rightModel(), msg, this.rightThread().threadId);
        }, 200);
      }

    } catch (e) {
      console.error('Error sending message:', e);
      this.sendLock.reset();
      this.status.set('Error sending message');
    }
  }

  private streamToChat(
    chat: 'left' | 'center' | 'right',
    provider: 'openai' | 'anthropic' | 'gemini',
    model: string,
    message: string,
    threadId: string | null
  ): void {
    // Prepare the contextual request with model-specific token configuration
    const request: StreamWithContextRequest = {
      threadId: threadId || undefined,
      message: message,
      model: model,
      provider: provider,
      context: {
        ...this.contextConfig,
        model: model  // Pass model for token-aware processing
      }
    };

    // Add empty assistant message for streaming
    const updateMessages = (chat === 'left' ? this.leftMessages :
      chat === 'center' ? this.centerMessages :
        this.rightMessages);

    const threadSignal = (chat === 'left' ? this.leftThread :
      chat === 'center' ? this.centerThread :
        this.rightThread);

    const assistantTimestamp = this.getTimestamp();
    updateMessages.update(ms => [
      ...ms,
      {
        role: 'assistant',
        content: '',
        timestamp: assistantTimestamp,
        isStreaming: true
      }
    ]);

    // Set streaming flag to show raw text instead of markdown during streaming
    const streamingSignal = chat === 'left' ? this.leftStreaming :
      chat === 'center' ? this.centerStreaming :
        this.rightStreaming;
    streamingSignal.set(true);

    this.debugLog(`💬 [${chat}] Starting stream for ${provider} with model ${model}`);
    this.debugLog(`💬 [${chat}] Request details:`, {
      threadId: request.threadId,
      messageLength: request.message?.length || 0,
      provider: request.provider,
      model: request.model
    });

    // Stream the response using global update queue
    let chunkCount = 0;
    this.api.streamWithContext(request).subscribe({
      next: (chunk: string) => {
        // Skip null, undefined, or empty chunks
        if (chunk === null || chunk === undefined || chunk === '') {
          return;
        }

        chunkCount++;
        if (environment.debugMode && chunkCount % 50 === 0) {
          console.log(`🎯 [${chat}] Received ${chunkCount} chunks so far...`);
        }

        // Add to global pending updates
        if (!this.pendingUpdates.has(chat)) {
          this.pendingUpdates.set(chat, []);
        }
        this.pendingUpdates.get(chat)!.push(chunk);

        // Schedule global flush if not already scheduled
        this.scheduleGlobalFlush();
      },
      error: (err) => {
        this.zone.run(() => {
          // Cancel any pending global flush timer
          if (this.updateTimerId !== null) {
            window.clearTimeout(this.updateTimerId);
            this.updateTimerId = null;
          }

          // Flush any pending chunks for this chat before showing error
          this.flushChatUpdates(chat, updateMessages, threadSignal);
          this.finalizeAssistantMessage(chat);

          if (!environment.production) {
            console.error(`💬 [${chat}] Stream error:`, err);
          }
          const errorMessage = this.formatErrorMessage(err);
          const formattedError = this.messageFormatter.formatMessage(errorMessage).content;
          updateMessages.update(ms => [
            ...ms,
            {
              role: 'assistant',
              content: errorMessage,
              timestamp: assistantTimestamp,
              formattedContent: formattedError,
              isStreaming: false
            }
          ]);
          this.status.set(`Error with ${chat} model`);

          // Clear streaming flag and scroll after error
          setTimeout(() => {
            streamingSignal.set(false);
            this.lastScrollTime.delete(chat); // Clear throttle for final scroll
            this.scrollToBottom(chat);
            this.finalizeSendLock(chat);
          }, 100);
        });
      },
      complete: () => {
        this.zone.run(() => {
          if (environment.debugMode) {
            console.log(`🎯 [${chat}] Observable COMPLETE event received for ${provider}`);
          }

          // Cancel any pending global flush timer
          if (this.updateTimerId !== null) {
            window.clearTimeout(this.updateTimerId);
            this.updateTimerId = null;
          }

          // Flush any pending chunks for this chat before finishing
          this.flushChatUpdates(chat, updateMessages, threadSignal);
          this.finalizeAssistantMessage(chat);

          this.debugLog(`💬 [${chat}] Stream completed for ${provider}`);

          // Get current content length before formatting
          const currentMessages = (chat === 'left' ? this.leftMessages() :
            chat === 'center' ? this.centerMessages() :
              this.rightMessages());
          const currentAssistantMessage = currentMessages.filter(m => m.role === 'assistant').pop();
          const streamedContentLength = currentAssistantMessage?.content?.length || 0;
          this.debugLog(`💬 [${chat}] Streamed content length before formatting: ${streamedContentLength}`);

          // Update threadId if backend provided one (either new thread or corrected existing)
          if (request.threadId && request.threadId !== threadId) {
            this.debugLog(`💬 [${chat}] Updating threadId from ${threadId} to ${request.threadId}`);
            const threadSignal = chat === 'left' ? this.leftThread :
              chat === 'center' ? this.centerThread :
                this.rightThread;
            threadSignal.update(t => ({ ...t, threadId: request.threadId || null }));
          }

          // Clear streaming flag first, then schedule highlight and scroll
          setTimeout(() => {
            if (environment.debugMode) {
              console.log(`🎯 [${chat}] Clearing streaming flag and rendering markdown`);
            }
            streamingSignal.set(false);

            // Clear scroll throttle so final scroll happens immediately
            this.lastScrollTime.delete(chat);

            // Now that markdown is rendered, schedule highlighting for code blocks
            this.scheduleHighlight(chat);
            this.scrollToBottom(chat);
            if (environment.debugMode) {
              console.log(`🎯 [${chat}] Completion handler finished`);
            }
            this.finalizeSendLock(chat);
          }, 100);
        });
      }
    });
  }

  private formatErrorMessage(error: any): string {
    return error?.message || 'An error occurred.';
  }

  private preprocessGeminiContent(content: string): string {
    // Pre-process Gemini content to handle HTML tags and entities
    let result = content
      // Preserve existing HTML tags by converting them to placeholders
      .replace(/<sup>([^<]+)<\/sup>/gi, '__SUP_START__$1__SUP_END__')
      .replace(/<sub>([^<]+)<\/sub>/gi, '__SUB_START__$1__SUB_END__');

    // Handle backticks - process line by line to avoid breaking across lines
    const lines = result.split('\n');
    result = lines.map(line => {
      return line.replace(/`([^`]+)`/g, (match, inner) => {
        // If it contains math-like expressions, treat the whole thing as math
        if (inner.includes('Enc(') ||
          inner.includes('Dec(') ||
          inner.includes('mod') ||
          inner.includes('*') ||
          inner.includes('+') ||
          inner.includes('-') ||
          inner.includes('/') ||
          inner.includes('=') ||
          inner.includes('^')) {
          // This is a full math expression, keep it as inline code
          return match;
        }
        // Check if it's a simple math variable
        else if (/^[a-zA-Z]$/.test(inner) ||
          /^[a-zA-Z]\d+$/.test(inner) ||
          /^[a-zA-Z]_\d+$/.test(inner) ||
          /^[nNmMcCqQLfg]_?\d*$/.test(inner) ||
          /^Q_L$/.test(inner)) {
          // Single variables get math formatting
          return `__MATH_VAR__${inner}__MATH_VAR_END__`;
        }
        return match; // Keep as code otherwise
      });
    }).join('\n');

    // Handle {q_l} style notation
    result = result.replace(/\{([a-zA-Z]_[a-zA-Z0-9]+)\}/g, '__MATH_VAR__{$1}__MATH_VAR_END__');

    return result;
  }

  private scrollToBottom(chat: 'left' | 'center' | 'right') {
    this.scrollPendingChats.set(chat, this.maxScrollAttempts);
    this.scheduleScrollFrame();
  }

  private scheduleScrollFrame() {
    if (this.scrollFrameId !== null || typeof window === 'undefined') {
      return;
    }

    const useAnimationFrame = typeof window.requestAnimationFrame === 'function';

    if (useAnimationFrame) {
      this.scrollFrameId = window.requestAnimationFrame(() => {
        this.scrollFrameId = null;
        this.scrollFrameCancel = null;
        this.flushScrollQueue();
      });
      this.scrollFrameCancel = () => {
        if (this.scrollFrameId !== null) {
          window.cancelAnimationFrame(this.scrollFrameId);
          this.scrollFrameId = null;
        }
      };
    } else {
      const timeoutId = window.setTimeout(() => {
        this.scrollFrameId = null;
        this.scrollFrameCancel = null;
        this.flushScrollQueue();
      }, 16);
      this.scrollFrameId = timeoutId;
      this.scrollFrameCancel = () => {
        window.clearTimeout(timeoutId);
        this.scrollFrameId = null;
      };
    }
  }

  private flushScrollQueue() {
    if (this.scrollPendingChats.size === 0) {
      return;
    }

    const currentBatch = new Map(this.scrollPendingChats);
    this.scrollPendingChats.clear();

    currentBatch.forEach((remainingAttempts, chat) => {
      this.performScroll(chat);
      if (remainingAttempts > 1) {
        const existing = this.scrollPendingChats.get(chat) ?? 0;
        this.scrollPendingChats.set(chat, Math.max(existing, remainingAttempts - 1));
      }
    });

    if (this.scrollPendingChats.size > 0) {
      this.scheduleScrollFrame();
    }
  }

  private scheduleHighlight(chat: 'left' | 'center' | 'right') {
    if (typeof window === 'undefined') {
      return;
    }
    this.highlightPendingChats.add(chat);
    if (this.highlightFrameId !== null) {
      return;
    }

    // Use 200ms delay to avoid overwhelming highlight.js with too many blocks
    const timeoutId = window.setTimeout(() => {
      this.highlightFrameId = null;
      this.highlightFrameCancel = null;
      this.applyPendingHighlights();
    }, 200);
    this.highlightFrameId = timeoutId;
    this.highlightFrameCancel = () => {
      window.clearTimeout(timeoutId);
      this.highlightFrameId = null;
    };
  }

  private applyPendingHighlights() {
    if (this.highlightPendingChats.size === 0) {
      return;
    }

    const chats = Array.from(this.highlightPendingChats);
    this.highlightPendingChats.clear();

    this.zone.run(() => {
      chats.forEach(chat => {
        const container = document.querySelector(`.chat-box.${chat} .messages`);
        if (!container) return;
        const codeBlocks = container.querySelectorAll('pre > code:not([data-highlighted])');

        // Limit to 10 blocks per batch to avoid overwhelming the browser
        const maxBlocksPerBatch = 10;
        let processed = 0;

        codeBlocks.forEach(block => {
          if (processed >= maxBlocksPerBatch) {
            // Schedule another highlight pass for remaining blocks
            this.scheduleHighlight(chat);
            return;
          }

          try {
            hljs.highlightElement(block as HTMLElement);
            block.setAttribute('data-highlighted', 'true');
            processed++;
          } catch (error) {
            console.error('Highlight error:', error);
            block.setAttribute('data-highlighted', 'true'); // Mark as attempted to avoid retry loop
          }
        });
      });
    });
  }

  private performScroll(chat: 'left' | 'center' | 'right') {
    const chatBox = document.querySelector('.chat-box.' + chat);
    if (!chatBox) return;
    const messages = chatBox.querySelector('.messages') as HTMLElement | null;
    if (!messages) return;
    if (messages.scrollHeight > messages.clientHeight) {
      messages.scrollTop = messages.scrollHeight;
    }
  }

  /**
   * Check if content needs backend formatting
   */
  private doesContentNeedFormatting(content: string): boolean {
    return /```/.test(content) || /\$\$.*\$\$/.test(content) || /<thinking>/.test(content);
  }

  // Setup listener for header new chat button
  private setupNewChatListener() {
    this.handleNewChatEvent = this.handleNewChatEvent.bind(this);
    this.handleToggleHistoryEvent = this.handleToggleHistoryEvent.bind(this);
    window.addEventListener('newChat', this.handleNewChatEvent);
    window.addEventListener('toggleHistory', this.handleToggleHistoryEvent);
  }

  private handleNewChatEvent = (event: CustomEvent) => {
    this.newChat();
  }

  private handleToggleHistoryEvent = (event: CustomEvent) => {
    if (event.detail?.forceShow) {
      // Force show history (coming from another page)
      this.showHistory.set(true);
      if (this.chatHistoryComponent) {
        setTimeout(() => {
          this.chatHistoryComponent?.loadThreads();
        }, 100);
      }
    } else {
      // Regular toggle
      this.toggleHistory();
    }
  }

  // Global update batching methods
  private scheduleGlobalFlush() {
    if (this.updateTimerId !== null) {
      return; // Already scheduled
    }

    this.updateTimerId = window.setTimeout(() => {
      this.zone.run(() => {
        this.flushAllPendingUpdates();
      });
    }, 150);
  }

  private flushAllPendingUpdates() {
    this.updateTimerId = null;

    if (this.pendingUpdates.size === 0) {
      return;
    }

    // Flush all chats' pending updates
    this.pendingUpdates.forEach((chunks, chat) => {
      if (chunks.length === 0) return;

      const combinedChunk = chunks.join('');
      const updateMessages = chat === 'left' ? this.leftMessages :
        chat === 'center' ? this.centerMessages :
          this.rightMessages;

      const threadSignal = chat === 'left' ? this.leftThread :
        chat === 'center' ? this.centerThread :
          this.rightThread;

      updateMessages.update(ms => {
        const copy = ms.slice();
        const last = copy[copy.length - 1];
        if (last && last.role === 'assistant') {
          last.content = (last.content || '') + combinedChunk;
        }
        return copy;
      });

      threadSignal.update(t => {
        const msgs = [...t.messages];
        if (msgs.length > 0 && msgs[msgs.length - 1].role === 'assistant') {
          msgs[msgs.length - 1].content += combinedChunk;
        } else {
          msgs.push({ role: 'assistant', content: combinedChunk, timestamp: this.getTimestamp() });
        }
        return { ...t, messages: msgs };
      });

      // Skip highlight scheduling during streaming - raw text doesn't need it
      // Throttle scroll to bottom - only every 500ms to reduce layout recalculations
      this.throttledScrollToBottom(chat);
    });

    this.pendingUpdates.clear();
  }

  private throttledScrollToBottom(chat: 'left' | 'center' | 'right') {
    const now = Date.now();
    const lastScroll = this.lastScrollTime.get(chat) || 0;

    // Only scroll if 500ms has passed since last scroll
    if (now - lastScroll >= 500) {
      this.scrollToBottom(chat);
      this.lastScrollTime.set(chat, now);
    }
  }

  private flushChatUpdates(
    chat: 'left' | 'center' | 'right',
    updateMessages: any,
    threadSignal: any
  ) {
    const chunks = this.pendingUpdates.get(chat);
    if (!chunks || chunks.length === 0) return;

    const combinedChunk = chunks.join('');
    this.pendingUpdates.delete(chat);

    updateMessages.update(ms => {
      const copy = ms.slice();
      const last = copy[copy.length - 1];
      if (last && last.role === 'assistant') {
        last.content = (last.content || '') + combinedChunk;
      }
      return copy;
    });

    threadSignal.update(t => {
      const msgs = [...t.messages];
      if (msgs.length > 0 && msgs[msgs.length - 1].role === 'assistant') {
        msgs[msgs.length - 1].content += combinedChunk;
      } else {
        msgs.push({ role: 'assistant', content: combinedChunk, timestamp: this.getTimestamp() });
      }
      return { ...t, messages: msgs };
    });

    // Don't schedule highlight or scroll here - called by completion handler which does it
  }

  private finalizeAssistantMessage(chat: 'left' | 'center' | 'right') {
    const messageSignal = chat === 'left' ? this.leftMessages :
      chat === 'center' ? this.centerMessages :
        this.rightMessages;

    messageSignal.update(messages => {
      if (!messages.length) {
        return messages;
      }

      const copy = messages.slice();
      for (let i = copy.length - 1; i >= 0; i--) {
        const msg = copy[i];
        if (msg.role !== 'assistant') {
          continue;
        }

        if (!msg.isStreaming) {
          break;
        }

        msg.isStreaming = false;
        msg.formattedContent = this.messageFormatter.formatMessage(msg.content).content;
        break;
      }

      return copy;
    });
  }

  async newChat() {
    try {
      this.status.set('Starting new conversation...');
      
      // Clear all messages
      this.leftMessages.set([]);
      this.centerMessages.set([]);
      this.rightMessages.set([]);
      
      // Reset all threads
      this.leftThread.set({ threadId: null, messages: [] });
      this.centerThread.set({ threadId: null, messages: [] });
      this.rightThread.set({ threadId: null, messages: [] });
      
      // Clear user input
      this.userInput.set('');
      
      this.status.set('New conversation started!');
      
      // Clear status after a moment
      setTimeout(() => {
        this.status.set('');
      }, 2000);
      
    } catch (error) {
      console.error('Error starting new chat:', error);
      this.status.set('Error starting new conversation');
    }
  }
}
