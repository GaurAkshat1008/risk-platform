import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import keycloak from "./keycloak";
import App from "./App";
import "./styles/global.css";

keycloak
  .init({ onLoad: "login-required", checkLoginIframe: false })
  .then((authenticated) => {
    if (!authenticated) {
      keycloak.login();
      return;
    }
    const root = createRoot(document.getElementById("root")!);
    root.render(
      <StrictMode>
        <App />
      </StrictMode>,
    );
  })
  .catch((err) => {
    console.error("Keycloak init failed", err);
    document.body.innerHTML =
      '<div style="display:flex;align-items:center;justify-content:center;height:100vh;font-family:sans-serif;color:#888">Authentication service unavailable</div>';
  });
