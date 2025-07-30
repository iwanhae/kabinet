import React from "react";
import {
  Typography,
  Box,
  Grid,
  Card,
  CardContent,
  styled,
} from "@mui/material";
import EventsTimelineChart from "../components/EventsTimelineChart";
import TopNoisyNamespaces from "../components/TopNoisyNamespaces";
import TopWarningReasons from "../components/TopWarningReasons";
import RecentCriticalEvents from "../components/RecentCriticalEvents";
import MetricsOverview from "../components/MetricsOverview";

const HeaderBox = styled(Box)({
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
  marginBottom: "2rem",
});

const Insight: React.FC = () => {
  return (
    <Box style={{ width: "100%" }}>
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
      <MetricsOverview />

      {/* Problem Analysis Section */}
      <Box sx={{ mt: 6 }}>
        <Typography variant="h5" gutterBottom sx={{ fontWeight: 600, mb: 3 }}>
          Problem Analysis
        </Typography>

        <Grid container spacing={3}>
          <Grid size={{ xs: 12, md: 4 }}>
            <TopNoisyNamespaces />
          </Grid>

          <Grid size={{ xs: 12, md: 4 }}>
            <TopWarningReasons />
          </Grid>

          <Grid size={{ xs: 12, md: 4 }}>
            <RecentCriticalEvents />
          </Grid>
        </Grid>
      </Box>
    </Box>
  );
};

export default Insight;
