import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "./styles.css";
import { applyAppearance, getAppearance } from "./appearance";

applyAppearance(document.documentElement, getAppearance());

const root = document.getElementById("root");
if (!root) throw new Error("应用入口不存在");

ReactDOM.createRoot(root).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
