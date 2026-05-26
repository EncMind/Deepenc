import { Component, OnInit, OnDestroy, signal, ViewChild, ElementRef, NgZone, WritableSignal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';
import { MarkdownModule } from 'ngx-markdown';
import { ChatService, EnhancedMessage } from '../chat.service';
import { SessionService, FreeSession } from '../services/session.service';
import { MessageFormatterService, FormattedMessage } from '../services/message-formatter.service';
import { MarkdownConfigService } from '../services/markdown-config.service';
import { MessageSendLock, ChatPanel } from '../shared/message-send-lock';
import { Subscription } from 'rxjs';
import { environment } from '../../environments/environment';

interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
  timestamp: string;
  formattedContent?: string;
  isStreaming?: boolean;
}

@Component({
  selector: 'app-free-chat',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterModule, MarkdownModule],
  template: `
    <div class="free-chat-container">
      <div class="main-content">
        <!-- Chat Controls -->
        <div class="chat-controls" *ngIf="!leftChatOpen() || !centerChatOpen() || !rightChatOpen()">
          <button class="chat-control-button"
                  (click)="expandAllChats()"
                  *ngIf="!leftChatOpen() && !centerChatOpen() && !rightChatOpen()">
            Open All Chats
          </button>
          <button class="chat-control-button"
                  (click)="toggleLeftChat()"
                  *ngIf="!leftChatOpen()">
            Open ChatGPT
          </button>
          <button class="chat-control-button"
                  (click)="toggleCenterChat()"
                  *ngIf="!centerChatOpen()">
            Open Claude
          </button>
          <button class="chat-control-button"
                  (click)="toggleRightChat()"
                  *ngIf="!rightChatOpen()">
            Open Gemini
          </button>
        </div>

        <!-- Chat Interface - Exactly like main app -->
        <div class="chat-interface"
             [class.single-chat]="(leftChatOpen() && !centerChatOpen() && !rightChatOpen()) || (!leftChatOpen() && centerChatOpen() && !rightChatOpen()) || (!leftChatOpen() && !centerChatOpen() && rightChatOpen())"
             [class.dual-chat]="(leftChatOpen() && centerChatOpen() && !rightChatOpen()) || (leftChatOpen() && !centerChatOpen() && rightChatOpen()) || (!leftChatOpen() && centerChatOpen() && rightChatOpen())">

          <!-- Left Chat (ChatGPT) -->
          <div class="chat-box left" *ngIf="leftChatOpen()">
            <button class="close-button" (click)="toggleLeftChat()" aria-label="Close ChatGPT">×</button>
            <div class="chat-header">
              <div class="chat-title-container">
                <img src="assets/chatgpt-logo.webp" alt="ChatGPT" class="chat-logo">
                <span class="chat-title">ChatGPT</span>
              </div>
            </div>

            <div class="chat-model-row">
              <label class="sr-only" for="freeLeftModel">ChatGPT model</label>
              <select id="freeLeftModel"
                      class="chat-model-select"
                      aria-label="ChatGPT model"
                      disabled>
                <option selected>GPT-4o Mini</option>
              </select>
            </div>

            <div class="messages" #leftMessagesContainer>
              <div *ngFor="let message of leftMessages()" class="message"
                   [class.user-message]="message.role === 'user'"
                   [class.assistant-message]="message.role === 'assistant'">
                <div class="message-header">
                  <span class="role">
                    <ng-container *ngIf="message.role === 'user'">👤 You</ng-container>
                    <ng-container *ngIf="message.role === 'assistant'">
                      <img src="assets/chatgpt-logo.webp" alt="ChatGPT" class="message-logo">
                      ChatGPT
                    </ng-container>
                  </span>
                  <div class="message-actions">
                    <span class="timestamp">{{ message.timestamp }}</span>
                    <button class="copy-message-button" 
                            (click)="copyMessage(message.content, $event)"
                            title="Copy message">
                      📋 Copy
                    </button>
                  </div>
                </div>
                <div class="content">
                  <ng-container *ngIf="message.role === 'assistant'; else leftUserPlain">
                    <ng-container *ngIf="!message.isStreaming; else leftStreaming">
                      <markdown [data]="message.formattedContent || message.content"
                               [options]="markdownOptions"
                               ngPreserveWhitespaces></markdown>
                    </ng-container>
                    <ng-template #leftStreaming>
                      <pre class="streaming-content">{{ message.content }}</pre>
                    </ng-template>
                  </ng-container>
                  <ng-template #leftUserPlain>
                    <div class="user-plain-text" [innerText]="message.content"></div>
                  </ng-template>
                </div>
              </div>
            </div>
          </div>

          <!-- Center Chat (Claude) -->
          <div class="chat-box center" *ngIf="centerChatOpen()">
            <button class="close-button" (click)="toggleCenterChat()" aria-label="Close Claude">×</button>
            <div class="chat-header">
              <div class="chat-title-container">
                <img src="assets/claude-logo.png" alt="Claude" class="chat-logo">
                <span class="chat-title">Claude</span>
              </div>
            </div>

            <div class="chat-model-row">
              <label class="sr-only" for="freeCenterModel">Claude model</label>
              <select id="freeCenterModel"
                      class="chat-model-select"
                      aria-label="Claude model"
                      disabled>
                <option selected>Claude 3 Haiku</option>
              </select>
            </div>

            <div class="messages" #centerMessagesContainer>
              <div *ngFor="let message of centerMessages()" class="message"
                   [class.user-message]="message.role === 'user'"
                   [class.assistant-message]="message.role === 'assistant'">
                <div class="message-header">
                  <span class="role">
                    <ng-container *ngIf="message.role === 'user'">👤 You</ng-container>
                    <ng-container *ngIf="message.role === 'assistant'">
                      <img src="assets/claude-logo.png" alt="Claude" class="message-logo">
                      Claude
                    </ng-container>
                  </span>
                  <div class="message-actions">
                    <span class="timestamp">{{ message.timestamp }}</span>
                    <button class="copy-message-button" 
                            (click)="copyMessage(message.content, $event)"
                            title="Copy message">
                      📋 Copy
                    </button>
                  </div>
                </div>
                <div class="content">
                  <ng-container *ngIf="message.role === 'assistant'; else centerUserPlain">
                    <ng-container *ngIf="!message.isStreaming; else centerStreaming">
                      <markdown [data]="message.formattedContent || message.content"
                               [options]="markdownOptions"
                               ngPreserveWhitespaces></markdown>
                    </ng-container>
                    <ng-template #centerStreaming>
                      <pre class="streaming-content">{{ message.content }}</pre>
                    </ng-template>
                  </ng-container>
                  <ng-template #centerUserPlain>
                    <div class="user-plain-text" [innerText]="message.content"></div>
                  </ng-template>
                </div>
              </div>
            </div>
          </div>

          <!-- Right Chat (Gemini) -->
          <div class="chat-box right" *ngIf="rightChatOpen()">
            <button class="close-button" (click)="toggleRightChat()" aria-label="Close Gemini">×</button>
            <div class="chat-header">
              <div class="chat-title-container">
                <img src="assets/gemini-logo.jpeg" alt="Gemini" class="chat-logo">
                <span class="chat-title">Gemini</span>
              </div>
            </div>

            <div class="chat-model-row">
              <label class="sr-only" for="freeRightModel">Gemini model</label>
              <select id="freeRightModel"
                      class="chat-model-select"
                      aria-label="Gemini model"
                      disabled>
                <option selected>Gemini 2.0 Flash Lite</option>
              </select>
            </div>

            <div class="messages" #rightMessagesContainer>
              <div *ngFor="let message of rightMessages()" class="message"
                   [class.user-message]="message.role === 'user'"
                   [class.assistant-message]="message.role === 'assistant'">
                <div class="message-header">
                  <span class="role">
                    <ng-container *ngIf="message.role === 'user'">👤 You</ng-container>
                    <ng-container *ngIf="message.role === 'assistant'">
                      <img src="assets/gemini-logo.jpeg" alt="Gemini" class="message-logo">
                      Gemini
                    </ng-container>
                  </span>
                  <div class="message-actions">
                    <span class="timestamp">{{ message.timestamp }}</span>
                    <button class="copy-message-button" 
                            (click)="copyMessage(message.content, $event)"
                            title="Copy message">
                      📋 Copy
                    </button>
                  </div>
                </div>
                <div class="content">
                  <ng-container *ngIf="message.role === 'assistant'; else rightUserPlain">
                    <ng-container *ngIf="!message.isStreaming; else rightStreaming">
                      <markdown [data]="message.formattedContent || message.content"
                               [options]="markdownOptions"
                               ngPreserveWhitespaces></markdown>
                    </ng-container>
                    <ng-template #rightStreaming>
                      <pre class="streaming-content">{{ message.content }}</pre>
                    </ng-template>
                  </ng-container>
                  <ng-template #rightUserPlain>
                    <div class="user-plain-text" [innerText]="message.content"></div>
                  </ng-template>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Input Area - TEST -->
        <div class="input-container">
          <textarea
            placeholder="Type your message..."
            [value]="userInput()"
            (input)="onUserInputChange($event)"
            (keydown.enter)="onKeyDown($event)"
            [disabled]="isLoading()">
          </textarea>
          <button class="send-button" 
                  (click)="sendMessage()"
                  [disabled]="!userInput().trim() || isLoading()">
            <span *ngIf="!isLoading()">Send</span>
            <span *ngIf="isLoading()">...</span>
          </button>
        </div>

        <!-- Limit Reached -->
        <div class="limit-reached" *ngIf="isLimitReached()">
          <div class="limit-content">
            <h3>Free messages used up!</h3>
            <p>Create an account to earn your tokens and continue the conversations with conversation history and access to all AI models.</p>
            <div class="limit-actions">
              <a routerLink="/auth" class="upgrade-button">Create Account</a>
            </div>
          </div>
        </div>

        <!-- Usage Counter -->
        <div class="usage-counter" *ngIf="getRemainingMessages() > 0">
          <span>{{ getRemainingMessages() }} messages remaining</span>
        </div>

        <div class="offline-warning" *ngIf="offlineWarning()">
          {{ offlineWarning() }}
        </div>

        <!-- Security Note -->
        <div class="security-note">
          <p>🔒 Your conversations are end-to-end encrypted and protected by hardware security.</p>
        </div>
      </div>
    </div>
  `,
  styleUrls: ['../app.component.css', './free-chat.component.css']
})
export class FreeChatComponent implements OnInit, OnDestroy {
  @ViewChild('leftMessagesContainer') leftMessagesContainer?: ElementRef<HTMLDivElement>;
  @ViewChild('centerMessagesContainer') centerMessagesContainer?: ElementRef<HTMLDivElement>;
  @ViewChild('rightMessagesContainer') rightMessagesContainer?: ElementRef<HTMLDivElement>;

  leftMessages = signal<ChatMessage[]>([]);
  centerMessages = signal<ChatMessage[]>([]);
  rightMessages = signal<ChatMessage[]>([]);
  leftChatOpen = signal<boolean>(true);
  centerChatOpen = signal<boolean>(true);
  rightChatOpen = signal<boolean>(true);
  
  userInput = signal<string>('');
  sendLock = new MessageSendLock();
  isLoading = computed(() => this.sendLock.locked());
  offlineWarning = signal<string>('');
  private cachedAnonymousUserId: string | null = null;
  
  // Fixed models for free tier
  private leftModel = 'gpt-4o-mini';     // OpenAI free tier
  private centerModel = 'claude-3-haiku'; // Anthropic free tier
  private rightModel = 'gemini-2.0-flash-lite'; // Google free tier
  
  // Session tracking  
  private session: FreeSession;
  private subscriptions = new Subscription();
  private maxMessages = 15;
  private streamBuffers = new Map<'left' | 'center' | 'right', string[]>();
  private streamFlushTimer: number | null = null;
  private debug(...args: unknown[]): void {
    if (!environment.debugMode || !args.length) return;
    // Use console to avoid recursive calls and stack overflow
    console.debug('[FreeChat]', ...args);
  }

  // Markdown configuration
  markdownOptions: any;

  constructor(
    private chatService: ChatService,
    private sessionService: SessionService,
    private zone: NgZone,
    private messageFormatter: MessageFormatterService,
    private markdownConfig: MarkdownConfigService
  ) {
    // Initialize with a default session, will be loaded in ngOnInit
    this.session = {
      messageCount: 0,
      startTime: Date.now(),
      dailyUsage: { date: new Date().toISOString().split('T')[0], count: 0 },
      leftSessionId: null,
      centerSessionId: null,
      rightSessionId: null,
      anonymousUserId: this.generateAnonymousUserId()
    };
  }

  async ngOnInit() {
    // Initialize markdown configuration
    this.markdownOptions = this.markdownConfig.getMarkedOptions();
    this.markdownConfig.initializeAll();

    this.session = await this.loadSession();
    await this.restoreChatsFromSession();
    this.debug('🔍 Free Chat initialized with session:', {
      leftSessionId: this.session.leftSessionId,
      centerSessionId: this.session.centerSessionId,
      rightSessionId: this.session.rightSessionId,
      anonymousUserId: this.session.anonymousUserId,
      messageCount: this.session.messageCount
    });
  }

  // Streaming rendering helpers
  hasIncompleteCodeBlock(content: string): boolean {
    if (!content) return false;
    const openBlocks = (content.match(/```/g) || []).length;
    return openBlocks % 2 !== 0;
  }

  private escapeHtml(s: string): string {
    return s
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');
  }

  formatRawStreamingContent(content: string): string {
    const escaped = this.escapeHtml(content || '');
    return `<pre class="streaming-content">${escaped}</pre>`;
  }

  private async loadSession(): Promise<FreeSession> {
    const today = this.getTodayIso();
    
    this.debug('🔍 loadSession - loading from SessionService');
    
    try {
      const session = await this.sessionService.getSession(this.generateAnonymousUserId());
      this.debug('🔍 loadSession - got session from service:', session);
      this.offlineWarning.set('');
      
      // Check for daily reset
      if (session.dailyUsage.date !== today) {
        this.debug('🔍 loadSession - daily reset needed, old date:', session.dailyUsage.date, 'new date:', today);
        session.dailyUsage = { date: today, count: 0 };
        session.dailyUsageTimestamp = Date.now();
        session.messageCount = 0;
        // DO NOT reset session IDs on daily reset
        
        // Save the reset session
        await this.sessionService.saveSession(session);
      }
      
      this.debug('🔍 loadSession - returning session:', session);
      return session;
    } catch (error) {
      console.error('❌ loadSession - error loading session:', error);
      
      // Fallback to default session
      this.offlineWarning.set('Working offline. Changes will sync when the connection returns.');
      const newSession: FreeSession = {
        messageCount: 0,
        startTime: Date.now(),
        dailyUsage: { date: today, count: 0 },
        leftSessionId: null,
        centerSessionId: null,
        rightSessionId: null,
        anonymousUserId: this.generateAnonymousUserId(),
        lastUpdated: Date.now(),
        dailyUsageTimestamp: Date.now()
      };
      this.debug('🔍 loadSession - created fallback session:', newSession);
      return newSession;
    }
  }

  private async restoreChatsFromSession() {
    const tasks: Promise<void>[] = [];

    if (this.session.leftSessionId) {
      tasks.push(this.loadThreadMessagesIntoChat('left', this.session.leftSessionId));
    }
    if (this.session.centerSessionId) {
      tasks.push(this.loadThreadMessagesIntoChat('center', this.session.centerSessionId));
    }
    if (this.session.rightSessionId) {
      tasks.push(this.loadThreadMessagesIntoChat('right', this.session.rightSessionId));
    }

    if (!tasks.length) {
      return;
    }

    const results = await Promise.allSettled(tasks);
    results.forEach((result, index) => {
      if (result.status === 'rejected') {
        console.error('❌ restoreChatsFromSession - failed to restore chat', index, result.reason);
      }
    });
  }

  private async loadThreadMessagesIntoChat(chat: 'left' | 'center' | 'right', threadId: string) {
    try {
      const response = await this.chatService.getThreadMessages(threadId);
      const enhancedMessages = response?.messages || [];
      if (!enhancedMessages.length) {
        return;
      }

      const restoredMessages = enhancedMessages.map(msg => this.mapEnhancedToChatMessage(msg));

      const signal = chat === 'left' ? this.leftMessages :
        chat === 'center' ? this.centerMessages : this.rightMessages;

      signal.set(restoredMessages);
      this.updateSessionThreadId(chat, threadId);
      this.scrollToBottom(chat);
    } catch (error) {
      console.error(`[FreeChat] Failed to restore ${chat} chat from thread ${threadId}:`, error);
      const signal = chat === 'left' ? this.leftMessages :
        chat === 'center' ? this.centerMessages : this.rightMessages;
      signal.set([]);
      this.updateSessionThreadId(chat, null);
    }
  }

  private mapEnhancedToChatMessage(msg: EnhancedMessage): ChatMessage {
    const role: 'user' | 'assistant' = msg.role === 'assistant' ? 'assistant' : 'user';
    const timestamp = msg.createdAt
      ? new Date(msg.createdAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
      : new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

    const chatMessage: ChatMessage = {
      role,
      content: msg.content || '',
      timestamp,
      isStreaming: false
    };

    if (role === 'assistant') {
      chatMessage.formattedContent = this.messageFormatter.formatMessage(chatMessage.content).content;
    }

    return chatMessage;
  }

  private async saveSession() {
    this.debug('🔍 saveSession - saving session:', this.session);
    try {
      await this.sessionService.saveSession(this.session);
      this.debug('🔍 saveSession - saved to SessionService');
      this.offlineWarning.set('');
    } catch (error) {
      console.error('❌ saveSession - error saving session:', error);
      this.offlineWarning.set('Unable to sync with server. Changes will be saved when you reconnect.');
    }
  }

  getRemainingMessages(): number {
    return Math.max(0, this.maxMessages - this.session.messageCount);
  }

  isLimitReached(): boolean {
    return this.session.messageCount >= this.maxMessages;
  }

  toggleLeftChat() {
    this.leftChatOpen.update(open => !open);
  }

  toggleCenterChat() {
    this.centerChatOpen.update(open => !open);
  }

  toggleRightChat() {
    this.rightChatOpen.update(open => !open);
  }

  expandAllChats() {
    this.leftChatOpen.set(true);
    this.centerChatOpen.set(true);
    this.rightChatOpen.set(true);
  }

  onUserInputChange(event: any) {
    this.userInput.set(event.target.value);
  }

  onKeyDown(event: KeyboardEvent) {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      if (this.sendLock.isLocked()) {
        return;
      }
      this.sendMessage();
    }
  }

  private getOpenChats(): ChatPanel[] {
    const chats: ChatPanel[] = [];
    if (this.leftChatOpen()) {
      chats.push('left');
    }
    if (this.centerChatOpen()) {
      chats.push('center');
    }
    if (this.rightChatOpen()) {
      chats.push('right');
    }
    return chats;
  }

  async sendMessage() {
    if (!this.userInput().trim() || this.sendLock.isLocked() || this.isLimitReached()) {
      return;
    }

    const openChats = this.getOpenChats();
    if (!openChats.length) {
      return;
    }

    const userMessage = this.userInput().trim();
    const messageCost = 1; // All free tier models cost 1 message
    
    if (this.session.messageCount + messageCost > this.maxMessages) {
      this.session.messageCount = this.maxMessages;
      this.saveSession();
      return;
    }

    // Add user message to all chats
    const timestamp = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    const userMsg: ChatMessage = {
      role: 'user',
      content: userMessage,
      timestamp
    };

    this.leftMessages.update(msgs => [...msgs, userMsg]);
    this.centerMessages.update(msgs => [...msgs, userMsg]);
    this.rightMessages.update(msgs => [...msgs, userMsg]);

    // Update session
    const todayIso = this.getTodayIso();
    this.session.messageCount += messageCost;
    this.session.dailyUsage = this.session.dailyUsage || { date: todayIso, count: 0 };
    if (this.session.dailyUsage.date !== todayIso) {
      this.session.dailyUsage.date = todayIso;
      this.session.dailyUsage.count = 0;
    }
    this.session.dailyUsage.count += messageCost;
    const now = Date.now();
    this.session.dailyUsageTimestamp = now;
    this.session.lastUpdated = now;
    this.saveSession();

    // Clear input and start loading
    this.userInput.set('');
    this.sendLock.begin(openChats);

    const streamPromises: Promise<void>[] = [];

    if (openChats.includes('left')) {
      streamPromises.push(this.streamToChat('left', 'openai', this.leftModel, userMessage));
    }
    if (openChats.includes('center')) {
      streamPromises.push(this.streamToChat('center', 'anthropic', this.centerModel, userMessage));
    }
    if (openChats.includes('right')) {
      streamPromises.push(this.streamToChat('right', 'gemini', this.rightModel, userMessage));
    }

    // Start all active streams
    try {
      if (streamPromises.length > 0) {
        await Promise.all(streamPromises);
      }
    } catch (error) {
      console.error('Error sending message:', error);
    }
  }

  private async streamToChat(
    chat: 'left' | 'center' | 'right',
    provider: 'openai' | 'anthropic' | 'gemini',
    model: string,
    message: string
  ) {
    // Get the persistent session ID for this chat
    const sessionId = chat === 'left' ? this.session.leftSessionId :
                     chat === 'center' ? this.session.centerSessionId : this.session.rightSessionId;
    
    const streamRequest = {
      threadId: sessionId ? sessionId : undefined, // Use persistent session as thread ID, null becomes undefined
      userId: this.session.anonymousUserId, // Anonymous user ID for free tier requests
      message: message,
      model: model,
      provider: provider,
      context: {
        ragEnabled: true,
        strategy: 'hybrid' as const,
        model: model
      }
    };

    this.debug(`🚀 Sending ${chat} request with threadId:`, sessionId, 'request:', streamRequest);

    const messagesSignal = chat === 'left' ? this.leftMessages :
      chat === 'center' ? this.centerMessages : this.rightMessages;

    const assistantTimestamp = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    messagesSignal.update(msgs => [
      ...msgs,
      {
        role: 'assistant',
        content: '',
        timestamp: assistantTimestamp,
        isStreaming: true
      }
    ]);
    
    return new Promise<void>((resolve, reject) => {
      const releaseLock = () => this.sendLock.release(chat);
      try {
        const sub = this.chatService.streamWithContext(streamRequest).subscribe({
          next: (chunk: string) => {
            this.zone.run(() => {
              if (!chunk) return;
              this.enqueueStreamChunk(chat, chunk);
            });
          },
          error: (error: any) => {
            this.zone.run(() => {
              console.error(`Stream error for ${chat}:`, error);
              this.flushStreamBuffers(chat);
              this.finalizeStreamingMessage(chat, messagesSignal);
              const errorMessage = 'Sorry, I encountered an error. Please try again.';
              const formattedError = this.messageFormatter.formatMessage(errorMessage).content;
              messagesSignal.update(msgs => [
                ...msgs,
                {
                  role: 'assistant',
                  content: errorMessage,
                  timestamp: assistantTimestamp,
                  formattedContent: formattedError,
                  isStreaming: false
                }
              ]);
              releaseLock();
              reject(error);
            });
          },
          complete: () => {
            this.zone.run(() => {
              this.flushStreamBuffers(chat);
              this.finalizeStreamingMessage(chat, messagesSignal);

              // Update threadId if backend provided one (either new thread or corrected existing)
              // The chatService updates streamRequest.threadId when it gets X-Thread-ID header
              if (streamRequest.threadId && streamRequest.threadId !== sessionId) {
                this.debug(`[${chat}] 🔧 Updating threadId from ${sessionId} to ${streamRequest.threadId}`);
                this.updateSessionThreadId(chat, streamRequest.threadId);
              }
              this.scrollToBottom(chat);
              releaseLock();
              resolve();
            });
          }
        });
        sub.add(releaseLock);
        this.subscriptions.add(sub);
      } catch (err) {
        releaseLock();
        reject(err);
      }
    });
  }

  private enqueueStreamChunk(chat: 'left' | 'center' | 'right', chunk: string) {
    if (!chunk) return;
    if (!this.streamBuffers.has(chat)) {
      this.streamBuffers.set(chat, []);
    }
    this.streamBuffers.get(chat)!.push(chunk);
    this.scheduleStreamFlush();
  }

  private scheduleStreamFlush() {
    if (this.streamFlushTimer !== null) {
      return;
    }
    if (typeof window === 'undefined') {
      this.flushStreamBuffers();
      return;
    }
    this.streamFlushTimer = window.setTimeout(() => {
      this.streamFlushTimer = null;
      this.flushStreamBuffers();
    }, 120);
  }

  private flushStreamBuffers(targetChat?: 'left' | 'center' | 'right') {
    const chatsToFlush = targetChat ? [targetChat] : Array.from(this.streamBuffers.keys());

    chatsToFlush.forEach(chat => {
      const chunks = this.streamBuffers.get(chat);
      if (!chunks || chunks.length === 0) {
        this.streamBuffers.delete(chat);
        return;
      }

      const combinedChunk = chunks.join('');
      this.streamBuffers.delete(chat);

      const messagesSignal = chat === 'left' ? this.leftMessages :
        chat === 'center' ? this.centerMessages : this.rightMessages;

      messagesSignal.update(msgs => {
        if (!msgs.length) return msgs;
        const copy = [...msgs];
        const last = copy[copy.length - 1];
        if (last && last.role === 'assistant') {
          last.content = (last.content || '') + combinedChunk;
        }
        return copy;
      });

      this.scrollToBottom(chat);
    });

    if (this.streamBuffers.size === 0 && this.streamFlushTimer !== null) {
      if (typeof window !== 'undefined') {
        window.clearTimeout(this.streamFlushTimer);
      }
      this.streamFlushTimer = null;
    }
  }

  private finalizeStreamingMessage(
    chat: 'left' | 'center' | 'right',
    messagesSignal: WritableSignal<ChatMessage[]>
  ) {
    messagesSignal.update(msgs => {
      if (!msgs.length) return msgs;
      const copy = [...msgs];
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

  private scrollToBottom(chat: 'left' | 'center' | 'right') {
    const container = chat === 'left' ? this.leftMessagesContainer :
      chat === 'center' ? this.centerMessagesContainer : this.rightMessagesContainer;
    
    if (container) {
      const element = container.nativeElement;
      element.scrollTop = element.scrollHeight;
    }
  }

  copyMessage(content: string, event?: Event) {
    const button = event?.target as HTMLElement;
    const original = button?.textContent || '📋 Copy';

    const tryAsync = async () => {
      if (navigator.clipboard && (window as any).isSecureContext) {
        await navigator.clipboard.writeText(content);
        return true;
      }
      return false;
    };

    const tryTextarea = () => {
      const ta = document.createElement('textarea');
      ta.value = content;
      ta.readOnly = true;
      ta.style.position = 'fixed';
      ta.style.top = '0';
      ta.style.left = '-9999px';
      document.body.appendChild(ta);
      ta.select();
      ta.setSelectionRange(0, ta.value.length);
      let ok = false;
      try { ok = document.execCommand('copy'); } catch { ok = false; }
      document.body.removeChild(ta);
      return ok;
    };

    const showSuccess = () => {
      if (button) {
        button.textContent = '✅ Copied!';
        setTimeout(() => button.textContent = original, 1800);
      }
    };

    const showFail = () => {
      if (button) {
        button.textContent = '❌ Failed';
        setTimeout(() => button.textContent = original, 1800);
      }
    };

    tryAsync().then(ok => ok ? showSuccess() : (tryTextarea() ? showSuccess() : showFail()))
             .catch(() => tryTextarea() ? showSuccess() : showFail());
  }

  getFormattedMessage(content: string): string {
    if (!content) return '';
    const formatted = this.messageFormatter.formatMessage(content);
    return formatted.content;
  }

  private updateSessionThreadId(chat: 'left' | 'center' | 'right', newThreadId: string | null) {
    this.debug(`🔧 Updating ${chat} session threadId to:`, newThreadId);
    if (chat === 'left') {
      this.session.leftSessionId = newThreadId;
    } else if (chat === 'center') {
      this.session.centerSessionId = newThreadId;
    } else if (chat === 'right') {
      this.session.rightSessionId = newThreadId;
    }
    this.saveSession();
  }

  resetSession() {
    const baseTime = Date.now();
    const todayIso = this.getTodayIso();
    const anonymousId = this.generateAnonymousUserId();
    this.session = {
      messageCount: 0,
      startTime: baseTime,
      dailyUsage: { date: todayIso, count: 0 },
      leftSessionId: null,
      centerSessionId: null,
      rightSessionId: null,
      anonymousUserId: anonymousId
    };
    this.saveSession();
    this.leftMessages.set([]);
    this.centerMessages.set([]);
    this.rightMessages.set([]);
  }

  private generateAnonymousUserId(): string {
    if (this.cachedAnonymousUserId) {
      return this.cachedAnonymousUserId;
    }
    const stored = this.getStoredAnonymousUserId();
    if (stored) {
      this.cachedAnonymousUserId = stored;
      return stored;
    }

    const timestamp = Date.now();
    const random = Math.random().toString(36).substring(2, 15);
    const anonymousUserId = `anon_${timestamp}_${random}`;

    this.storeAnonymousUserId(anonymousUserId);

    return anonymousUserId;
  }

  private getStoredAnonymousUserId(): string | null {
    try {
      if (typeof window !== 'undefined') {
        const value = window.localStorage.getItem('deepenc_anonymous_user_id');
        if (value) return value;
      }
    } catch {
      // ignore storage errors
    }
    if (typeof document !== 'undefined') {
      const match = document.cookie.match(/(?:^|; )deepenc_anonymous_user_id=([^;]+)/);
      if (match) return decodeURIComponent(match[1]);
    }
    return null;
  }

  private storeAnonymousUserId(id: string): void {
    try {
      if (typeof window !== 'undefined') {
        window.localStorage.setItem('deepenc_anonymous_user_id', id);
      }
    } catch {
      // ignore storage errors
    }
    if (typeof document !== 'undefined') {
      const expires = new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toUTCString();
      const secure = typeof window !== 'undefined' && window.location?.protocol === 'https:' ? '; Secure' : '';
      document.cookie = `deepenc_anonymous_user_id=${encodeURIComponent(id)}; path=/; expires=${expires}; SameSite=Lax${secure}`;
    }
    this.cachedAnonymousUserId = id;
  }

  private getTodayIso(): string {
    return new Date().toISOString().split('T')[0];
  }

  ngOnDestroy(): void {
    this.subscriptions.unsubscribe();
    if (this.streamFlushTimer !== null && typeof window !== 'undefined') {
      window.clearTimeout(this.streamFlushTimer);
    }
    this.streamFlushTimer = null;
    this.streamBuffers.clear();
    this.sendLock.reset();
  }
}
