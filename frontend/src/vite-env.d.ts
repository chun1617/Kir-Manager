/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

// Wails runtime bindings
import type * as WailsApp from '../wailsjs/go/main/App'
import type { main, kiroprocess } from '../wailsjs/go/models'

interface WailsRuntime {
  EventsOn(eventName: string, callback: (...data: unknown[]) => void): () => void
  EventsOff(eventName: string): void
  EventsEmit(eventName: string, ...data: unknown[]): void
}

declare global {
  interface Window {
    go: {
      main: {
        App: typeof WailsApp
      }
    }
    runtime: WailsRuntime
  }
}
