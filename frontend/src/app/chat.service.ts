import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { AuthService } from './auth.service';
import { BillingService } from './services/billing.service';
import { environment } from '../environments/environment';

export interface ChatMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp?: string;
}

export interface ChatRequest {
  model: string;
  messages: ChatMessage[];
}

export interface ModelsResponse {
  openai: string[];
  anthropic: string[];
  gemini: string[];
  defaultOpenAI: string;
  defaultAnthropic: string;
  defaultGemini: string;
}

export interface ChosenModelsResponse {
  openai: string;
  anthropic: string;
  gemini: string;
}

export interface SetChosenModelsRequest {
  openai: string;
  anthropic: string;
  gemini: string;
}

export interface StreamWithContextRequest {
  threadId?: string;
  userId?: string;
  message: string;
  model: string;
  provider: string;
  context: {
    maxMessages?: number;     // Deprecated: kept for backwards compatibility
    ragCount?: number;        // Deprecated: kept for backwards compatibility
    ragEnabled?: boolean;
    strategy?: 'recent' | 'rag' | 'hybrid';
    model?: string;           // Model name for token-aware processing
    tokenBudget?: number;     // Override default token budget
    useTokenLimits?: boolean; // Use token-aware truncation instead of message count
  };
}

export interface Thread {
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

export interface EnhancedMessage {
  id: string;
  type: string;
  threadId: string;
  userId: string;
  role: string;
  content: string;
  provider: string;
  model: string;
  createdAt: number;
  vectorId?: string;
}

type Provider = 'openai' | 'anthropic' | 'gemini';

@Injectable({ providedIn: 'root' })
export class ChatService {
  private base = environment.apiUrl || '/api';
  private authService = inject(AuthService);
  private billingService = inject(BillingService);
  private modelsCache: ModelsResponse | null = null;
  private modelsPromise: Promise<ModelsResponse> | null = null;
  private modelsUserId: string | null = null;
  private debug(...args: unknown[]): void {
    if (!environment.debugMode) return;
    if (!args.length) return;
    console.debug('[ChatService]', ...args);
  }

  async getModels(): Promise<ModelsResponse> {
    const currentUserId = this.authService.getCurrentUser()?.uid || 'anonymous';
    if (this.modelsUserId !== currentUserId) {
      // User changed—drop cached models so tier-specific filtering is correct
      this.modelsCache = null;
      this.modelsPromise = null;
      this.modelsUserId = currentUserId;
    }

    // Return cached version if available
    if (this.modelsCache) {
      return this.modelsCache;
    }

    // Return existing promise if already fetching
    if (this.modelsPromise) {
      return this.modelsPromise;
    }

    // Create new promise for fetching
    this.modelsPromise = this.fetchModels();
    
    try {
      this.modelsCache = await this.modelsPromise;
      return this.modelsCache;
    } finally {
      this.modelsPromise = null; // Reset promise after completion
    }
  }

  private async fetchModels(): Promise<ModelsResponse> {
    const headers = await this.getAuthHeaders();

    const res = await fetch(`${this.base}/models`, { 
      method: 'GET',
      headers
    });
    if (!res.ok) {
      throw new Error(`Models request failed: ${res.status}`);
    }
    return res.json();
  }


  // Check if a model requires upgrade
  isModelRestrictedForCurrentTier(model: string): boolean {
    return this.billingService.isUpgradeRequired(model, this.getCurrentTier());
  }

  // Get current user tier
  getCurrentTier(): string {
    const subscription = this.billingService.getCachedSubscriptionInfo();
    if (subscription?.tier) {
      return subscription.tier;
    }

    const usage = this.billingService.getCachedUsageStats();
    if (usage?.currentTier) {
      return usage.currentTier;
    }

    return 'free';
  }

  // Get provider for a model
  getModelProvider(model: string): string {
    return this.billingService.getModelProvider(model);
  }

  // Check if user can make a request
  canMakeRequest(estimatedTokens: number): boolean {
    return this.billingService.canMakeRequest(estimatedTokens);
  }

  // Estimate tokens for a message (simple heuristic)
  estimateTokens(text: string): number {
    // Simple estimation: ~4 characters per token
    return Math.ceil(text.length / 4);
  }

  // Get model display name with tier info
  getModelDisplayNameWithTierInfo(model: string): string {
    const baseDisplayName = this.getModelDisplayName(model);
    if (this.isModelRestrictedForCurrentTier(model)) {
      return `${baseDisplayName} (Upgrade Required)`;
    }
    return baseDisplayName;
  }

  // Get model display name (existing method, just for reference)
  getModelDisplayName(model: string): string {
    const displayNames: { [key: string]: string } = {
      // OpenAI models
      'gpt-4o': 'GPT-4o',
      'gpt-4o-mini': 'GPT-4o Mini',
      'gpt-5.1': 'GPT-5.1',
      'gpt-5-mini': 'GPT-5 Mini',
      
      // Anthropic models
      'claude-3-haiku': 'Claude 3 Haiku',
      'claude-sonnet-4-5': 'Claude Sonnet 4.5',
      'claude-opus-4-5': 'Claude Opus 4.5',
      
      // Gemini models
      'gemini-3-pro-preview': 'Gemini 3 Pro Preview',
      'gemini-2.5-flash': 'Gemini 2.5 Flash',
      'gemini-2.0-flash-lite': 'Gemini 2.0 Flash Lite',
      'gemini-1.0-pro': 'Gemini 1.0 Pro'
    };
    
    return displayNames[model] || model;
  }

  async getChosenModels(): Promise<ChosenModelsResponse> {
    const headers = await this.getAuthHeaders();

    const res = await fetch(`${this.base}/chosen-models`, { 
      method: 'GET',
      headers
    });
    if (!res.ok) {
      throw new Error(`Get chosen models failed: ${res.status}`);
    }
    return res.json();
  }

  async setChosenModels(chosenModels: SetChosenModelsRequest): Promise<void> {
    this.debug('setChosenModels called with:', JSON.stringify(chosenModels));
    
    const headers = await this.getAuthHeaders();
    headers['Content-Type'] = 'application/json';

    // Sanitize headers before logging to avoid leaking tokens
    const sanitizedHeaders = { ...headers } as Record<string, string | undefined>;
    if (sanitizedHeaders['Authorization']) sanitizedHeaders['Authorization'] = 'Bearer ***';

    this.debug('Making POST request to:', `${this.base}/chosen-models`);
    this.debug('Headers:', sanitizedHeaders);
    this.debug('Body:', JSON.stringify(chosenModels));

    const res = await fetch(`${this.base}/chosen-models`, { 
      method: 'POST',
      headers,
      body: JSON.stringify(chosenModels)
    });
    
    this.debug('Response status:', res.status);
    
    if (!res.ok) {
      // Do not log response body to avoid leaking any sensitive server details
      console.error('[ChatService] Error response status:', res.status);
      throw new Error(`Set chosen models failed: ${res.status}`);
    }
    
    this.debug('setChosenModels completed successfully');
  }

  async createThread(title: string): Promise<Thread> {
    const headers = await this.getAuthHeaders();
    headers['Content-Type'] = 'application/json';

    const res = await fetch(`${this.base}/threads`, {
      method: 'POST',
      headers,
      body: JSON.stringify({ title })
    });
    
    if (!res.ok) {
      // Do not read or include response body in error
      throw new Error(`Create thread failed: ${res.status}`);
    }
    
    return res.json();
  }

  async getThreads(): Promise<{ threads: Thread[], count: number }> {
    const headers = await this.getAuthHeaders();

    const res = await fetch(`${this.base}/threads`, {
      method: 'GET',
      headers
    });
    if (!res.ok) {
      throw new Error(`Get threads failed: ${res.status}`);
    }
    return res.json();
  }

  async getThreadMessages(threadId: string): Promise<{ messages: EnhancedMessage[], count: number }> {
    const headers = await this.getAuthHeaders();

    const url = `${this.base}/threads/${threadId}/messages`;
    this.debug('Fetching messages from URL:', url);
    // Sanitize headers before logging
    const sanitizedHeaders = { ...headers } as Record<string, string | undefined>;
    if (sanitizedHeaders['Authorization']) sanitizedHeaders['Authorization'] = 'Bearer ***';
    this.debug('Headers:', sanitizedHeaders);

    const res = await fetch(url, {
      method: 'GET',
      headers,
      cache: 'no-cache'
    });
    
    this.debug('Response status:', res.status);
    this.debug('Response content-type:', res.headers.get('content-type'));
    
    if (!res.ok) {
      // Avoid logging or reading response body which may include message data
      console.error('[ChatService] Error response status:', res.status);
      throw new Error(`Get messages failed: ${res.status}`);
    }
    
    const responseText = await res.text();
    // Do not log response body to avoid logging chat content
    try {
      return JSON.parse(responseText);
    } catch (error) {
      console.error('[ChatService] JSON parse error:', error);
      throw new Error('Invalid JSON response');
    }
  }

  /**
   * Stream with context using the new contextual endpoint
   */
  streamWithContext(request: StreamWithContextRequest): Observable<string> {
    const url = `${this.base}/stream/contextual`;

    return new Observable<string>((observer) => {
      let reader: ReadableStreamDefaultReader<Uint8Array> | null = null;
      let isCancelled = false;

      (async () => {
        try {
          const headers = await this.getAuthHeaders();
          headers['Content-Type'] = 'application/json';

          const res = await fetch(url, {
            method: 'POST',
            headers,
            body: JSON.stringify(request)
          });

          if (!res.ok || !res.body) {
            // Do not read or include response body in error
            throw new Error(`Stream HTTP ${res.status}`);
          }

          // Read X-Thread-ID header and update request threadId for history tracking
          const threadId = res.headers.get('X-Thread-ID');
          if (threadId && threadId !== request.threadId) {
            this.debug('Updating threadId from header:', threadId);
            request.threadId = threadId;
          }

          reader = res.body.getReader();
          const decoder = new TextDecoder('utf-8', { fatal: false });

          try {
            while (true) {
              // Check if cancelled before reading
              if (isCancelled) {
                await reader.cancel();
                break;
              }

              const { done, value } = await reader.read();
              if (done) {
                // Process any remaining bytes in the decoder
                const finalChunk = decoder.decode(new Uint8Array(0), { stream: false });
                if (finalChunk) {
                  observer.next(finalChunk);
                }
                break;
              }

              if (value && value.length > 0) {
                try {
                  const chunk = decoder.decode(value, { stream: true });
                  if (chunk && chunk.length > 0) {
                    observer.next(chunk);
                  }
                } catch (error) {
                  console.error('[ChatService] Decode error:', error);
                  // Continue processing rather than stopping
                }
              }
            }
          } finally {
            // Always release the reader properly
            if (reader) {
              try {
                reader.releaseLock();
              } catch (releaseError) {
                console.error('[ChatService] Error releasing reader:', releaseError);
              }
            }
          }

          if (!isCancelled) {
            observer.complete();
          }
        } catch (err) {
          // Clean up reader on error
          if (reader) {
            try {
              if (!isCancelled) {
                reader.cancel();
              }
              reader.releaseLock();
            } catch (releaseError) {
              console.error('[ChatService] Error cleaning up reader:', releaseError);
            }
          }
          if (!isCancelled) {
            observer.error(err);
          }
        }
      })();

      // Cleanup function for subscription cancellation
      return () => {
        isCancelled = true;
        // Note: We set the flag and let the reading loop handle cancellation
        // to avoid calling cancel() on an already released reader
      };
    });
  }

  /**
   * Legacy stream method for backward compatibility
   * Now internally uses the contextual endpoint
   */
  stream(provider: Provider, req: ChatRequest): Observable<string> {
    // Convert to contextual request
    const contextRequest: StreamWithContextRequest = {
      message: req.messages[req.messages.length - 1]?.content || '',
      model: req.model,
      provider: provider,
      context: {
        ragEnabled: true,
        strategy: 'hybrid',
        model: req.model        // Enable token-aware processing (default when model provided)
      }
    };

    return this.streamWithContext(contextRequest);
  }

  getCurrentUserId(): string | null {
    const user = this.authService.getCurrentUser();
    return user?.uid || null;
  }

  isAuthenticated(): boolean {
    return this.authService.isLoggedIn();
  }

  // Enhanced thread methods for history feature
  async getThreadsWithPagination(page: number = 1, limit: number = 20, search?: string): Promise<ThreadsResponse> {
    const params = new URLSearchParams({
      page: page.toString(),
      limit: limit.toString(),
    });
    
    if (search) {
      params.set('search', search);
    }
    
    const headers = await this.getAuthHeaders();
    const res = await fetch(`${this.base}/threads?${params}`, { 
      method: 'GET', 
      headers 
    });
    
    if (!res.ok) {
      throw new Error(`Get threads failed: ${res.status}`);
    }
    
    return res.json();
  }
  
  async renameThread(threadId: string, newTitle: string): Promise<void> {
    const headers = await this.getAuthHeaders();
    headers['Content-Type'] = 'application/json';
    
    const res = await fetch(`${this.base}/threads/${threadId}/rename`, {
      method: 'PUT',
      headers,
      body: JSON.stringify({ title: newTitle })
    });
    
    if (!res.ok) {
      throw new Error(`Rename thread failed: ${res.status}`);
    }
  }
  
  async deleteThread(threadId: string): Promise<void> {
    const headers = await this.getAuthHeaders();
    
    const res = await fetch(`${this.base}/threads/${threadId}/delete`, {
      method: 'DELETE', 
      headers
    });
    
    if (!res.ok) {
      throw new Error(`Delete thread failed: ${res.status}`);
    }
  }
  
  private async getAuthHeaders(): Promise<Record<string, string>> {
    const headers: Record<string, string> = {};
    const token = await this.authService.getIdToken();
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
      return headers;
    }

    const anonymousId = this.getAnonymousUserId();
    if (anonymousId) {
      headers['X-Anonymous-User-ID'] = anonymousId;
    }
    return headers;
  }

  private getAnonymousUserId(): string | null {
    if (typeof window === 'undefined') {
      return null;
    }
    try {
      return localStorage.getItem('deepenc_anonymous_user_id');
    } catch {
      return null;
    }
  }
}

interface ThreadsResponse {
  threads: Thread[];
  count: number;
  page: number;
  limit: number;
  hasMore: boolean;
}
