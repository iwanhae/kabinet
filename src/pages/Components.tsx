import React from "react";
import DimensionPage from "../components/dimension/DimensionPage";

/**
 * Grouped by COALESCE(source.component, reportingComponent) — controller
 * events (events.k8s.io) only set reportingComponent.
 */
const Components: React.FC = () => (
  <DimensionPage field="component" noun="Component" nounPlural="components" />
);

export default Components;
