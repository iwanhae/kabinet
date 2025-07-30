import React from "react";
import {
  Typography,
  Box,
  Grid,
  Card,
  CardContent,
  styled,
} from "@mui/material";
import { useLocation } from "wouter";
import MetricCard from "../components/MetricCard";
import { SmallChartPlaceholder } from "../components/ChartPlaceholder";
import EventsTimelineChart from "../components/EventsTimelineChart";

const HeaderBox = styled(Box)({
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
  marginBottom: "2rem",
});

const Insight: React.FC = () => {
  const [, setLocation] = useLocation();

  const metrics = [
    {
      title: "Total Events",
      value: "4.2M",
      change: -12,
      subtitle: "Last 7 days",
    },
    {
      title: "Error Rate",
      value: "14.3%",
      change: 28,
      subtitle: "Critical events ratio",
    },
    {
      title: "Failed Pods",
      value: "6,117",
      change: -35,
      subtitle: "FailedScheduling events",
    },
    {
      title: "Restarts",
      value: "28K",
      change: 32,
      subtitle: "Container restarts",
    },
    {
      title: "New Warnings",
      value: "97K",
      change: 91,
      subtitle: "Warning events",
    },
    {
      title: "Node Issues",
      value: "20K",
      change: 36,
      subtitle: "Node-related events",
    },
    {
      title: "Network Events",
      value: "290K",
      change: 67,
      subtitle: "Network-related events",
    },
    {
      title: "Storage Events",
      value: "17K",
      change: 101,
      subtitle: "Storage-related events",
    },
  ];

  const handleMetricClick = (title: string) => {
    // Navigate to Discover page with a sample query based on the metric
    const queries: Record<string, string> = {
      "Total Events":
        "SELECT * FROM $events ORDER BY metadata.creationTimestamp DESC LIMIT 100",
      "Error Rate":
        "SELECT * FROM $events WHERE type = 'Warning' OR type = 'Error' ORDER BY metadata.creationTimestamp DESC",
      "Failed Pods":
        "SELECT * FROM $events WHERE reason = 'FailedScheduling' ORDER BY metadata.creationTimestamp DESC",
      Restarts:
        "SELECT * FROM $events WHERE reason LIKE '%Restart%' ORDER BY metadata.creationTimestamp DESC",
    };

    const query = queries[title] || "SELECT * FROM $events LIMIT 100";
    setLocation(`/discover?query=${encodeURIComponent(query)}`);
  };

  return (
    <Box>
      {/* Header */}
      <HeaderBox>
        <Box>
          <Typography variant="h4" gutterBottom sx={{ fontWeight: 700, mb: 1 }}>
            Analytics
          </Typography>
          <Typography variant="body1" color="text.secondary">
            Account overview
          </Typography>
        </Box>
      </HeaderBox>

      {/* Event Timeline Chart Placeholder */}
      <Card sx={{ mb: 4 }}>
        <CardContent sx={{ p: 3 }}>
          <EventsTimelineChart />
        </CardContent>
      </Card>

      {/* Metrics Grid */}
      <Grid container spacing={3}>
        {metrics.map((metric, index) => (
          <Grid size={{ xs: 12, sm: 6, md: 3 }} key={index}>
            <MetricCard
              title={metric.title}
              value={metric.value}
              change={metric.change}
              subtitle={metric.subtitle}
              onClick={() => handleMetricClick(metric.title)}
            />
          </Grid>
        ))}
      </Grid>

      {/* Additional Section */}
      <Box sx={{ mt: 6 }}>
        <Typography variant="h5" gutterBottom sx={{ fontWeight: 600, mb: 3 }}>
          Event Activity
        </Typography>

        <Grid container spacing={3}>
          <Grid size={{ xs: 12, md: 6 }}>
            <Card>
              <CardContent sx={{ p: 3 }}>
                <Typography variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
                  Top Event Reasons
                </Typography>
                <SmallChartPlaceholder>
                  <Typography variant="body2" color="text.secondary">
                    Top Reasons Chart Placeholder
                  </Typography>
                </SmallChartPlaceholder>
              </CardContent>
            </Card>
          </Grid>

          <Grid size={{ xs: 12, md: 6 }}>
            <Card>
              <CardContent sx={{ p: 3 }}>
                <Typography variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
                  Namespace Activity
                </Typography>
                <SmallChartPlaceholder>
                  <Typography variant="body2" color="text.secondary">
                    Namespace Heatmap Placeholder
                  </Typography>
                </SmallChartPlaceholder>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      </Box>
    </Box>
  );
};

export default Insight;
