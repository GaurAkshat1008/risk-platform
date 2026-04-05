import { ApolloProvider } from "@apollo/client";
import { ConfigProvider, App as AntApp, theme as antTheme } from "antd";
import { BrowserRouter } from "react-router-dom";
import client from "@/apollo";
import { lightTheme, darkTheme } from "@/theme";
import { useThemeMode } from "@/hooks/useThemeMode";
import { AppShell } from "@/components/layout/AppShell";
import { ErrorBoundary } from "@/components/common/ErrorBoundary";
import { AppRoutes } from "@/routes";

export default function App() {
  const { mode, toggle } = useThemeMode();
  const themeConfig = mode === "dark" ? darkTheme : lightTheme;

  return (
    <ApolloProvider client={client}>
      <ConfigProvider
        theme={{
          ...themeConfig,
          algorithm:
            mode === "dark"
              ? antTheme.darkAlgorithm
              : antTheme.defaultAlgorithm,
        }}
      >
        <AntApp>
          <BrowserRouter>
            <ErrorBoundary>
              <AppShell themeMode={mode} onToggleTheme={toggle}>
                <AppRoutes />
              </AppShell>
            </ErrorBoundary>
          </BrowserRouter>
        </AntApp>
      </ConfigProvider>
    </ApolloProvider>
  );
}
