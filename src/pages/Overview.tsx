import React from "react";
import { Link } from "wouter";
import TimelineHistogram from "../components/charts/TimelineHistogram";
import CabinetHeatmap from "../components/charts/CabinetHeatmap";
import KpiStrip from "../components/overview/KpiStrip";
import TopMovers from "../components/overview/TopMovers";
import TopNamespaces from "../components/overview/TopNamespaces";
import { useNavigation } from "../hooks/useNavigation";
import { Card } from "../ui";
import nsStyles from "../components/overview/TopNamespaces.module.css";
import styles from "./Overview.module.css";

const Overview: React.FC = () => {
  const navigate = useNavigation();

  return (
    <div className={styles.page}>
      <Card title="Events timeline">
        <TimelineHistogram height={240} />
      </Card>

      <KpiStrip />

      <div className={styles.split}>
        <Card title="The Cabinet — namespaces × time">
          <CabinetHeatmap />
        </Card>
        <div className={styles.stack}>
          <Card
            title="Top namespaces"
            actions={
              <Link
                href={navigate({ page: "namespaces" })}
                className={nsStyles.viewAll}
              >
                View all →
              </Link>
            }
          >
            <TopNamespaces />
          </Card>
          <Card title="Top movers vs previous period">
            <TopMovers />
          </Card>
        </div>
      </div>
    </div>
  );
};

export default Overview;
