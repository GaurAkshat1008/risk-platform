import { Avatar, Button, Dropdown, Space, Tag, Typography } from "antd";
import {
  LogoutOutlined,
  MoonOutlined,
  SunOutlined,
  UserOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import { useAuth } from "@/hooks/useAuth";
type ThemeMode = "light" | "dark";

interface Props {
  themeMode: ThemeMode;
  onToggleTheme: () => void;
}

export function Header({ themeMode, onToggleTheme }: Props) {
  const { user, logout } = useAuth();

  const items = [
    {
      key: "user",
      label: (
        <Space direction="vertical" size={0}>
          <Typography.Text strong>{user.username}</Typography.Text>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {user.email}
          </Typography.Text>
        </Space>
      ),
      disabled: true,
    },
    { type: "divider" as const },
    {
      key: "logout",
      label: "Sign out",
      icon: <LogoutOutlined />,
      danger: true,
      onClick: logout,
    },
  ];

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        height: "100%",
        padding: "0 24px",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <Typography.Text type="secondary" style={{ fontSize: 12 }} keyboard>
          ⌘K
        </Typography.Text>
        <Button
          type="text"
          size="small"
          icon={<SearchOutlined />}
          onClick={() => {
            window.dispatchEvent(
              new KeyboardEvent("keydown", { key: "k", metaKey: true }),
            );
          }}
        >
          Jump to…
        </Button>
      </div>

      <Space size="middle">
        {user.tenantId && (
          <Tag style={{ margin: 0, fontWeight: 500 }}>
            Tenant: {user.tenantId.slice(0, 8)}
          </Tag>
        )}

        <Button
          type="text"
          size="small"
          icon={themeMode === "dark" ? <SunOutlined /> : <MoonOutlined />}
          onClick={onToggleTheme}
          aria-label="Toggle theme"
        />

        <Dropdown menu={{ items }} trigger={["click"]} placement="bottomRight">
          <Avatar
            size={28}
            icon={<UserOutlined />}
            style={{ cursor: "pointer", backgroundColor: "#475569" }}
          />
        </Dropdown>
      </Space>
    </div>
  );
}
