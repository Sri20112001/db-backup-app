import React from 'react'
import { AlertTriangle } from 'lucide-react'

interface ErrorBoundaryProps {
  children: React.ReactNode
}

interface ErrorBoundaryState {
  error: Error | null
}

// Last-resort crash guard: a render exception anywhere below this boundary
// shows a recoverable panel instead of a blank app. Mounted in AppLayout
// around the page outlet.
class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error }
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error('UI crashed:', error, info.componentStack)
  }

  private handleRetry = () => this.setState({ error: null })

  private handleReload = () => window.location.reload()

  render() {
    const { error } = this.state
    if (!error) return this.props.children
    return (
      <div className="flex-1 min-h-0 flex items-center justify-center p-8">
        <div className="max-w-md w-full p-6 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm text-center">
          <div className="mx-auto w-11 h-11 rounded-full bg-error-container/60 text-error flex items-center justify-center">
            <AlertTriangle size={20} />
          </div>
          <h1 className="mt-3 text-[16px] font-semibold text-on-surface">Something went wrong</h1>
          <p className="mt-1 text-[13px] text-on-surface-variant">
            This view crashed. Your data is safe — try again or reload the page.
          </p>
          <p className="mt-2 font-mono text-[11px] text-outline break-all">
            {error.message}
          </p>
          <div className="mt-4 flex items-center justify-center gap-2">
            <button
              type="button"
              onClick={this.handleRetry}
              className="px-4 h-9 rounded-lg bg-surface-container-low text-on-surface text-[13px] font-medium hover:bg-surface-container-high transition-colors"
            >
              Try again
            </button>
            <button
              type="button"
              onClick={this.handleReload}
              className="px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors"
            >
              Reload app
            </button>
          </div>
        </div>
      </div>
    )
  }
}

export default ErrorBoundary
