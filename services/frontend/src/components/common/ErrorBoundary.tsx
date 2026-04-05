import { Component, type ReactNode } from "react";
import { Result, Button } from "antd";

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  render() {
    if (this.state.error) {
      if (this.props.fallback) return this.props.fallback;
      return (
        <Result
          status="error"
          title="Something went wrong"
          subTitle={this.state.error.message}
          extra={
            <Button onClick={() => this.setState({ error: null })}>
              Try again
            </Button>
          }
        />
      );
    }
    return this.props.children;
  }
}
