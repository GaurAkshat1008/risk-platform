import { useState } from "react";
import { Layout } from "antd";
import { Sidebar } from "./Sidebar";
import { Header } from "./Header";
import { CommandPalette } from "@/components/common/CommandPalette";

const { Content, Header: AntHeader } = Layout;

type ThemeMode = "light" | "dark";

interface Props {
  themeMode: ThemeMode;
  onToggleTheme: () => void;
  children: React.ReactNode;
}

export function AppShell({ themeMode, onToggleTheme, children }: Props) {
  const [collapsed, setCollapsed] = useState(false);
  const siderWidth = collapsed ? 56 : 220;

  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Sidebar collapsed={collapsed} onCollapse={setCollapsed} />

      <Layout
        style={{ marginLeft: siderWidth, transition: "margin-left 0.2s" }}
      >
        <AntHeader
          style={{
            padding: 0,
            height: 48,
            lineHeight: "48px",
            borderBottom: "1px solid var(--border-color, #dee2e6)",
            position: "sticky",
            top: 0,
            zIndex: 9,
          }}
        >
          <Header themeMode={themeMode} onToggleTheme={onToggleTheme} />
        </AntHeader>

        <Content style={{ margin: 16 }}>{children}</Content>
      </Layout>

      <CommandPalette />
    </Layout>
  );
}
