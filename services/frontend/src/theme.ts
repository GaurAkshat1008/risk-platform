import type { ThemeConfig } from 'antd';

/**
 * RiskCore design system — Institutional Slate/Steel palette.
 *
 * Dense data-first layouts. Serious, authoritative, finance-grade.
 * Primary anchored on steel-slate with sharp teal accents for
 * interactive elements. Tight border radii for a no-nonsense feel.
 */

const shared: Partial<ThemeConfig['token']> = {
  fontFamily: "'DM Sans', -apple-system, BlinkMacSystemFont, sans-serif",
  fontSize: 13,
  borderRadius: 4,
  borderRadiusSM: 2,
  borderRadiusLG: 6,
  controlHeight: 32,
  controlHeightSM: 24,
  controlHeightLG: 40,
  wireframe: false,
};

export const lightTheme: ThemeConfig = {
  token: {
    ...shared,
    colorPrimary: '#475569',
    colorInfo: '#475569',
    colorSuccess: '#16a34a',
    colorWarning: '#d97706',
    colorError: '#dc2626',
    colorBgContainer: '#ffffff',
    colorBgElevated: '#ffffff',
    colorBgLayout: '#f1f3f5',
    colorBorder: '#dee2e6',
    colorBorderSecondary: '#e9ecef',
    colorText: '#1c1e21',
    colorTextSecondary: '#5c636a',
    colorTextTertiary: '#868e96',
    colorTextQuaternary: '#adb5bd',
  },
  components: {
    Layout: {
      siderBg: '#1e293b',
      headerBg: '#ffffff',
      bodyBg: '#f1f3f5',
    },
    Menu: {
      darkItemBg: '#1e293b',
      darkItemColor: '#94a3b8',
      darkItemHoverColor: '#e2e8f0',
      darkItemSelectedBg: '#334155',
      darkItemSelectedColor: '#f8fafc',
    },
    Table: {
      headerBg: '#f8f9fa',
      headerColor: '#5c636a',
      rowHoverBg: '#f8f9fa',
      borderColor: '#e9ecef',
    },
    Card: {
      paddingLG: 16,
    },
  },
};

export const darkTheme: ThemeConfig = {
  token: {
    ...shared,
    colorPrimary: '#94a3b8',
    colorInfo: '#94a3b8',
    colorSuccess: '#22c55e',
    colorWarning: '#f59e0b',
    colorError: '#ef4444',
    colorBgContainer: '#1a1d24',
    colorBgElevated: '#21252d',
    colorBgLayout: '#0f1117',
    colorBorder: '#2d333b',
    colorBorderSecondary: '#21262d',
    colorText: '#e1e4e8',
    colorTextSecondary: '#8b949e',
    colorTextTertiary: '#6e7681',
    colorTextQuaternary: '#484f58',
  },
  components: {
    Layout: {
      siderBg: '#0d1117',
      headerBg: '#1a1d24',
      bodyBg: '#0f1117',
    },
    Menu: {
      darkItemBg: '#0d1117',
      darkItemColor: '#8b949e',
      darkItemHoverColor: '#e1e4e8',
      darkItemSelectedBg: '#1a1d24',
      darkItemSelectedColor: '#f0f6fc',
    },
    Table: {
      headerBg: '#21252d',
      headerColor: '#8b949e',
      rowHoverBg: '#21252d',
      borderColor: '#2d333b',
    },
    Card: {
      paddingLG: 16,
    },
  },
};
