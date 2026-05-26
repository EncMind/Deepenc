import { Component, EventEmitter, Output, signal, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ChatService } from '../chat.service';

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

interface ThreadGroup {
  title: string;
  threads: Thread[];
  startDate: Date;
  endDate: Date;
}

@Component({
  selector: 'app-chat-history',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="history-sidebar">
      <div class="history-header">
        <div class="header-text">
          <h3>Chat History</h3>
        </div>
        <button class="close-button" (click)="closeHistory()" aria-label="Close Chat History">×</button>
      </div>
      
      <div class="history-content">
        <div class="history-scroll">
        
          <!-- Loading indicator -->
          <div *ngIf="isLoading()" class="loading">
            Loading conversations...
          </div>
          
          <!-- Thread Groups -->
          <div *ngIf="!isLoading()" class="thread-groups">
            <div *ngFor="let group of threadGroups()" class="thread-group">
              <h4 class="group-title">{{ group.title }}</h4>
              <div class="threads">
                <div *ngFor="let thread of group.threads" 
                     class="thread-item"
                     [class.active]="thread.id === activeThreadId()"
                     (click)="selectThread(thread)">
                  <div class="thread-content">
                    <div class="thread-title">{{ thread.title }}</div>
                    <div class="thread-meta">
                      <span class="message-count">{{ thread.metadata.messageCount }} msgs</span>
                      <span class="last-model">{{ getModelIcon(thread.metadata.lastModel) }}</span>
                    </div>
                    <div class="thread-date">{{ formatDate(thread.updatedAt) }}</div>
                  </div>
                  <div class="thread-actions" (click)="$event.stopPropagation()">
                    <button (click)="renameThread(thread, $event)" title="Rename">
                      <img src="assets/edit.png" alt="Edit" class="action-icon">
                    </button>
                    <button (click)="deleteThread(thread, $event)" title="Delete">
                      <img src="assets/delete.png" alt="Delete" class="action-icon">
                    </button>
                  </div>
                </div>
              </div>
            </div>
            
            <!-- Load More -->
            <button *ngIf="hasMoreThreads() && !isLoading()" 
                    class="load-more" 
                    (click)="loadMoreThreads()"
                    [disabled]="isLoadingMore()">
              {{ isLoadingMore() ? 'Loading...' : 'Load More' }}
            </button>
          </div>
          
          <!-- Empty state -->
          <div *ngIf="!isLoading() && threadGroups().length === 0" class="empty-state">
            <p>No conversations found.</p>
          </div>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .history-sidebar {
      --color-primary: #4f46e5;
      --color-primary-soft: #eef2ff;
      --color-primary-strong: #312e81;
      --color-surface: rgba(255, 255, 255, 0.75);
      --color-surface-muted: rgba(255, 255, 255, 0.55);
      --color-border: rgba(148, 163, 184, 0.28);
      --color-divider: rgba(15, 23, 42, 0.08);
      --color-body: #0f172a;
      --color-muted: #64748b;
      width: 320px;
      background:
        radial-gradient(140% 140% at 15% 20%, rgba(99, 102, 241, 0.12), transparent 55%),
        radial-gradient(110% 110% at 85% 0%, rgba(14, 116, 144, 0.08), transparent 60%),
        #f8fafc;
      border-right: 1px solid var(--color-border);
      height: 100vh;
      position: fixed;
      left: 0;
      top: 60px;
      z-index: 100;
      transition: transform 0.3s ease;
      display: flex;
      flex-direction: column;
      font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      color: var(--color-body);
      box-shadow:
        0 24px 48px rgba(15, 23, 42, 0.15),
        inset 0 1px 0 rgba(255, 255, 255, 0.5);
      overflow: hidden;
      /* Remove heavy blurs to prevent GPU instability */
      /* backdrop-filter disabled intentionally */
      line-height: 1.65;
    }

    .history-header {
      padding: 1.5rem 1.75rem 1.25rem;
      border-bottom: 1px solid var(--color-divider);
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 1rem;
      flex-shrink: 0;
      position: relative;
      background: linear-gradient(180deg, rgba(255, 255, 255, 0.4), transparent);
    }

    .history-header::after {
      content: '';
      position: absolute;
      inset: auto 1.75rem -0.5rem 1.75rem;
      height: 1px;
      background: linear-gradient(90deg, transparent, var(--color-primary), transparent);
      opacity: 0.2;
    }

    .header-text {
      display: flex;
      flex-direction: column;
      gap: 0.35rem;
    }

    .eyebrow {
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
      font-size: 0.72rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.12em;
      color: var(--color-primary);
      opacity: 0.75;
    }

    .history-header h3 {
      margin: 0;
      font-size: 1.4rem;
      font-weight: 700;
      letter-spacing: -0.02em;
      color: var(--color-body);
      background: linear-gradient(135deg, var(--color-body), var(--color-muted));
      -webkit-background-clip: text;
      background-clip: text;
    }

    .close-button {
      background: var(--color-surface-muted);
      border: 1px solid var(--color-border);
      font-size: 1.2rem;
      font-weight: 300;
      color: var(--color-muted);
      cursor: pointer;
      padding: 0;
      border-radius: 999px;
      width: 34px;
      height: 34px;
      display: flex;
      align-items: center;
      justify-content: center;
      transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
      /* blur disabled via .reduced-effects global override for Chrome */
      box-shadow: 0 4px 12px rgba(15, 23, 42, 0.08);
    }

    .close-button:hover {
      transform: translateY(-1px);
      background: linear-gradient(135deg, rgba(255, 255, 255, 0.95), var(--color-primary-soft));
      border-color: var(--color-primary);
      color: var(--color-primary-strong);
      box-shadow:
        0 8px 24px rgba(79, 70, 229, 0.2),
        0 2px 8px rgba(15, 23, 42, 0.1);
    }

    .history-content {
      flex: 1;
      display: flex;
      flex-direction: column;
      overflow: hidden;
      padding: 1.25rem 1.75rem 1.75rem;
      gap: 1.25rem;
    }

    .history-subtitle {
      margin: 0;
      font-size: 0.88rem;
      color: var(--color-muted);
      line-height: 1.65;
      padding-top: 0.25rem;
    }

    .history-scroll {
      flex: 1;
      display: flex;
      flex-direction: column;
      overflow: hidden;
    }


    .loading {
      padding: 3rem 1.5rem;
      text-align: center;
      color: var(--color-muted);
      font-size: 0.9rem;
      font-weight: 500;
      background: var(--color-surface);
      border: 1px solid var(--color-border);
      border-radius: 20px;
      /* Remove blur inside panels as well */
      box-shadow:
        0 8px 24px rgba(15, 23, 42, 0.08),
        inset 0 1px 0 rgba(255, 255, 255, 0.7);
    }

    .thread-groups {
      flex: 1;
      overflow-y: auto;
      padding-bottom: 1rem;
      display: flex;
      flex-direction: column;
      gap: 1.5rem;
      scrollbar-width: thin;
      scrollbar-color: rgba(79, 70, 229, 0.3) transparent;
    }

    .thread-groups::-webkit-scrollbar {
      width: 6px;
    }

    .thread-groups::-webkit-scrollbar-track {
      background: transparent;
    }

    .thread-groups::-webkit-scrollbar-thumb {
      background: rgba(79, 70, 229, 0.45);
      border-radius: 999px;
    }

    .group-title {
      font-size: 0.72rem;
      font-weight: 600;
      color: var(--color-muted);
      margin: 0 0 0.75rem 0;
      text-transform: uppercase;
      letter-spacing: 0.14em;
      padding-left: 0.5rem;
      position: relative;
      opacity: 0.85;
    }

    .group-title::before {
      content: '';
      position: absolute;
      left: 0;
      top: 50%;
      transform: translateY(-50%);
      width: 3px;
      height: 12px;
      background: linear-gradient(180deg, var(--color-primary), transparent);
      border-radius: 999px;
    }

    .thread-item {
      padding: 1rem 1.1rem;
      border-radius: 18px;
      cursor: pointer;
      margin-bottom: 0.5rem;
      transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
      border: 1px solid var(--color-border);
      position: relative;
      background: var(--color-surface);
      box-shadow:
        0 4px 12px rgba(15, 23, 42, 0.05),
        inset 0 1px 0 rgba(255, 255, 255, 0.6);
      /* blur disabled via .reduced-effects global override for Chrome */
    }

    .thread-item:hover {
      transform: translateY(-2px);
      background: linear-gradient(135deg, rgba(255, 255, 255, 0.95), var(--color-surface));
      border-color: var(--color-primary);
      box-shadow:
        0 12px 32px rgba(79, 70, 229, 0.15),
        0 4px 16px rgba(15, 23, 42, 0.1),
        inset 0 1px 0 rgba(255, 255, 255, 0.8);
    }

    .thread-item.active {
      background: linear-gradient(135deg, var(--color-primary-soft), rgba(255, 255, 255, 0.95));
      border-color: var(--color-primary);
      box-shadow:
        0 8px 24px rgba(79, 70, 229, 0.25),
        inset 0 1px 0 rgba(255, 255, 255, 0.9);
    }

    .thread-title {
      font-weight: 600;
      font-size: 0.92rem;
      color: var(--color-body);
      margin-bottom: 0.35rem;
      line-height: 1.3;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      padding-right: 2.2rem; /* Space for actions */
    }

    .thread-meta {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 0.75rem;
      font-size: 0.76rem;
      color: var(--color-muted);
    }

    .thread-date {
      font-size: 0.68rem;
      color: rgba(15, 23, 42, 0.45);
      margin-top: 0.45rem;
      font-weight: 500;
      letter-spacing: 0.02em;
    }

    .thread-actions {
      position: absolute;
      right: 0.6rem;
      top: 0.6rem;
      opacity: 0;
      transition: opacity 0.2s ease;
      display: flex;
      gap: 0.35rem;
      background: rgba(255, 255, 255, 0.9);
      padding: 0.28rem;
      border-radius: 999px;
      border: 1px solid rgba(79, 70, 229, 0.16);
      box-shadow: 0 10px 20px rgba(15, 23, 42, 0.1);
      /* blur disabled via .reduced-effects global override for Chrome */
    }

    .thread-item:hover .thread-actions {
      opacity: 1;
    }

    .thread-actions button {
      background: none;
      border: 1px solid transparent;
      padding: 0.28rem;
      cursor: pointer;
      border-radius: 999px;
      font-size: 0.8rem;
      display: flex;
      align-items: center;
      justify-content: center;
      transition: all 0.2s ease;
    }

    .thread-actions button:hover {
      border-color: rgba(79, 70, 229, 0.2);
      background: rgba(79, 70, 229, 0.08);
    }

    .action-icon {
      width: 15px;
      height: 15px;
      object-fit: contain;
    }

    .load-more {
      width: 90%;
      margin: 0 auto 1.5rem;
      padding: 1rem 1.5rem;
      background: var(--color-surface);
      border: 1px solid var(--color-border);
      border-radius: 999px;
      cursor: pointer;
      font-size: 0.9rem;
      font-weight: 600;
      color: var(--color-primary);
      transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
      display: block;
      text-align: center;
      /* blur disabled via .reduced-effects global override for Chrome */
      box-shadow:
        0 4px 12px rgba(15, 23, 42, 0.06),
        inset 0 1px 0 rgba(255, 255, 255, 0.7);
    }

    .load-more:hover:not(:disabled) {
      transform: translateY(-1px);
      background: linear-gradient(135deg, var(--color-primary-soft), rgba(255, 255, 255, 0.95));
      border-color: var(--color-primary);
      color: var(--color-primary-strong);
      box-shadow:
        0 8px 24px rgba(79, 70, 229, 0.2),
        0 2px 8px rgba(15, 23, 42, 0.1),
        inset 0 1px 0 rgba(255, 255, 255, 0.9);
    }

    .load-more:disabled {
      opacity: 0.5;
      cursor: not-allowed;
      transform: none;
    }

    .empty-state {
      padding: 3rem 2rem;
      text-align: center;
      color: var(--color-muted);
      background: var(--color-surface);
      border-radius: 20px;
      border: 1px solid var(--color-border);
      backdrop-filter: blur(20px);
      -webkit-backdrop-filter: blur(20px);
      box-shadow:
        0 8px 24px rgba(15, 23, 42, 0.08),
        inset 0 1px 0 rgba(255, 255, 255, 0.7);
    }

    .empty-state p {
      margin: 0;
      font-size: 0.95rem;
      font-weight: 500;
      line-height: 1.65;
    }

    /* Mobile responsiveness */
    @media (max-width: 768px) {
      .history-sidebar {
        width: 100%;
        transform: translateX(-100%);
        border-right: none;
        border-bottom: 1px solid rgba(79, 70, 229, 0.12);
        box-shadow: none;
        border-radius: 0 0 28px 28px;
        top: 56px;
      }
      
      
      .history-sidebar.open {
        transform: translateX(0);
        box-shadow: 0 24px 40px rgba(15, 23, 42, 0.18);
      }

      .history-content {
        padding: 0 1.25rem 1.5rem;
      }

      .thread-groups {
        padding-right: 0.2rem;
      }
    }
  `]
})
export class ChatHistoryComponent implements OnInit {
  private chatService = inject(ChatService);

  threadGroups = signal<ThreadGroup[]>([]);
  activeThreadId = signal<string | null>(null);
  hasMoreThreads = signal(true);
  isLoadingMore = signal(false);
  isLoading = signal(true);
  currentPage = signal(1);

  @Output() threadSelected = new EventEmitter<Thread>();
  @Output() threadDeleted = new EventEmitter<string>();
  @Output() historyClosed = new EventEmitter<void>();


  ngOnInit() {
    this.loadThreads();
  }


  async loadThreads(page: number = 1, search?: string) {
    try {
      if (page === 1) {
        this.isLoading.set(true);
      } else {
        this.isLoadingMore.set(true);
      }

      const response = await this.chatService.getThreadsWithPagination(page, 20, search);
      
      if (page === 1) {
        // New search or initial load
        this.threadGroups.set(this.groupThreadsByDate(response.threads));
      } else {
        // Load more - append to existing threads
        const existingThreads = this.getAllThreadsFromGroups();
        const allThreads = [...existingThreads, ...response.threads];
        this.threadGroups.set(this.groupThreadsByDate(allThreads));
      }

      this.hasMoreThreads.set(response.hasMore);
      this.currentPage.set(page);
    } catch (error) {
      console.error('Error loading threads:', error);
    } finally {
      this.isLoading.set(false);
      this.isLoadingMore.set(false);
    }
  }


  async loadMoreThreads() {
    if (this.hasMoreThreads() && !this.isLoadingMore()) {
      const nextPage = this.currentPage() + 1;
      await this.loadThreads(nextPage);
    }
  }

  selectThread(thread: Thread) {
    this.activeThreadId.set(thread.id);
    this.threadSelected.emit(thread);
  }

  closeHistory() {
    this.historyClosed.emit();
  }

  async renameThread(thread: Thread, event: Event) {
    event.stopPropagation();
    
    const newTitle = prompt('Enter new conversation title:', thread.title);
    if (newTitle && newTitle.trim() !== thread.title) {
      try {
        await this.chatService.renameThread(thread.id, newTitle.trim());
        // Update local thread title
        this.updateThreadTitle(thread.id, newTitle.trim());
      } catch (error) {
        console.error('Error renaming thread:', error);
        alert('Failed to rename conversation. Please try again.');
      }
    }
  }

  async deleteThread(thread: Thread, event: Event) {
    event.stopPropagation();
    
    if (confirm(`Delete conversation "${thread.title}"? This cannot be undone.`)) {
      try {
        await this.chatService.deleteThread(thread.id);
        this.removeThreadFromGroups(thread.id);
        this.threadDeleted.emit(thread.id);
        
        // Clear active thread if it was deleted
        if (this.activeThreadId() === thread.id) {
          this.activeThreadId.set(null);
        }
      } catch (error) {
        console.error('Error deleting thread:', error);
        alert('Failed to delete conversation. Please try again.');
      }
    }
  }

  private groupThreadsByDate(threads: Thread[]): ThreadGroup[] {
    const groups: ThreadGroup[] = [];
    const now = new Date();
    
    // Define date ranges
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const yesterday = new Date(today.getTime() - 24 * 60 * 60 * 1000);
    const weekAgo = new Date(today.getTime() - 7 * 24 * 60 * 60 * 1000);
    
    // Sort threads by updatedAt descending (newest first)
    const sortedThreads = [...threads].sort((a, b) => b.updatedAt - a.updatedAt);
    
    // Group threads
    const todayThreads = sortedThreads.filter(t => new Date(t.updatedAt) >= today);
    const yesterdayThreads = sortedThreads.filter(t => {
      const date = new Date(t.updatedAt);
      return date >= yesterday && date < today;
    });
    const weekThreads = sortedThreads.filter(t => {
      const date = new Date(t.updatedAt);
      return date >= weekAgo && date < yesterday;
    });
    const olderThreads = sortedThreads.filter(t => new Date(t.updatedAt) < weekAgo);
    
    if (todayThreads.length > 0) {
      groups.push({ title: 'Today', threads: todayThreads, startDate: today, endDate: now });
    }
    if (yesterdayThreads.length > 0) {
      groups.push({ title: 'Yesterday', threads: yesterdayThreads, startDate: yesterday, endDate: today });
    }
    if (weekThreads.length > 0) {
      groups.push({ title: 'Last 7 days', threads: weekThreads, startDate: weekAgo, endDate: yesterday });
    }
    if (olderThreads.length > 0) {
      groups.push({ title: 'Older', threads: olderThreads, startDate: new Date(0), endDate: weekAgo });
    }
    
    return groups;
  }

  private getAllThreadsFromGroups(): Thread[] {
    const threads: Thread[] = [];
    for (const group of this.threadGroups()) {
      threads.push(...group.threads);
    }
    return threads;
  }

  private updateThreadTitle(threadId: string, newTitle: string) {
    const groups = this.threadGroups().map(group => ({
      ...group,
      threads: group.threads.map(thread => 
        thread.id === threadId ? { ...thread, title: newTitle } : thread
      )
    }));
    this.threadGroups.set(groups);
  }

  private removeThreadFromGroups(threadId: string) {
    const groups = this.threadGroups()
      .map(group => ({
        ...group,
        threads: group.threads.filter(thread => thread.id !== threadId)
      }))
      .filter(group => group.threads.length > 0); // Remove empty groups
    this.threadGroups.set(groups);
  }

  getModelIcon(model: string): string {
    if (model.includes('gpt') || model.includes('openai')) return 'ChatGPT';
    if (model.includes('claude') || model.includes('anthropic')) return 'Claude';
    if (model.includes('gemini') || model.includes('google')) return 'Gemini';
    return 'AI';
  }

  formatDate(timestamp: number): string {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    
    // Less than 1 hour ago
    if (diff < 60 * 60 * 1000) {
      const minutes = Math.floor(diff / (60 * 1000));
      return minutes === 0 ? 'Just now' : `${minutes}m ago`;
    }
    
    // Less than 24 hours ago
    if (diff < 24 * 60 * 60 * 1000) {
      const hours = Math.floor(diff / (60 * 60 * 1000));
      return `${hours}h ago`;
    }
    
    // Less than 7 days ago
    if (diff < 7 * 24 * 60 * 60 * 1000) {
      const days = Math.floor(diff / (24 * 60 * 60 * 1000));
      return `${days}d ago`;
    }
    
    // Older - show date
    return date.toLocaleDateString();
  }
}
