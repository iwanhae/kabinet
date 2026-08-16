import React from "react";
import DimensionPage from "../components/dimension/DimensionPage";

/**
 * Grouped by source.host: the node name on kubelet-emitted events.
 * Controller/scheduler events have no host and are intentionally excluded —
 * this tab is about node-scoped activity.
 */
const Nodes: React.FC = () => (
  <DimensionPage field="host" noun="Node" nounPlural="nodes" topN={100} />
);

export default Nodes;
