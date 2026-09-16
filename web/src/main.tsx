import "@fontsource-variable/inter";
import "@fontsource-variable/jetbrains-mono";
import { bootstrapAppearance } from "@/services/appearance-bootstrap";

// Keep the public film page independent of the workspace bundle; it bootstraps appearance itself.
if (/^\/welcome\/?$/.test(window.location.pathname)) void import("./welcome-application");
else void bootstrapAppearance().finally(() => import("./application"));
