import { Signal, WritableSignal, signal } from '@angular/core';

export type ChatPanel = 'left' | 'center' | 'right';

/**
 * Tracks outstanding chat responses so UI can block new sends until all finish.
 */
export class MessageSendLock {
  private readonly activeChats = new Set<ChatPanel>();
  private readonly lockedSignal: WritableSignal<boolean> = signal(false);

  readonly locked: Signal<boolean> = this.lockedSignal.asReadonly();

  /**
   * Mark the provided chats as pending. Duplicates are ignored.
   */
  begin(chats: ChatPanel[]): void {
    let changed = false;
    chats.forEach(chat => {
      if (!this.activeChats.has(chat)) {
        this.activeChats.add(chat);
        changed = true;
      }
    });
    if (changed) {
      this.lockedSignal.set(true);
    }
  }

  /**
   * Release a chat panel once its response has finished.
   */
  release(chat: ChatPanel): void {
    if (this.activeChats.delete(chat) && this.activeChats.size === 0) {
      this.lockedSignal.set(false);
    }
  }

  /**
   * Clear all pending chats. Useful on teardown.
   */
  reset(): void {
    if (this.activeChats.size) {
      this.activeChats.clear();
      this.lockedSignal.set(false);
    }
  }

  /**
   * Convenience helper for class consumers.
   */
  isLocked(): boolean {
    return this.lockedSignal();
  }
}
