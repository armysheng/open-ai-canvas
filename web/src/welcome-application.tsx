import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import WelcomePage from "@/pages/welcome";
import { bootstrapAppearance } from "@/services/appearance-bootstrap";

void bootstrapAppearance();
createRoot(document.getElementById("root")!).render(<StrictMode><WelcomePage /></StrictMode>);
