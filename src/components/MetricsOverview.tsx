import React from "react";
import { Grid, CircularProgress, Box } from "@mui/material";
import { useQueryParams } from "../hooks/useUrlParams";
import MetricCard from "./MetricCard";
import { useEventsQuery } from "../hooks/useEventsQuery";

// Interface definitions for each metric query result
interface TotalEventsResult {
  total_count: number;
}

interface ErrorRateResult {
  warning_count: number;
  total_count: number;
}

interface FailedPodsResult {
  failed_pods_count: number;
}

interface RestartsResult {
  restarts_count: number;
}

interface NodeIssuesResult {
  node_issues_count: number;
}

interface NetworkEventsResult {
  network_events_count: number;
}

interface StorageEventsResult {
  storage_events_count: number;
}

const MetricsOverview: React.FC = () => {
  const { setQuery, setWhereClause } = useQueryParams();

  // Query for Total Events
  const totalEventsQuery = `SELECT COUNT(*) as total_count FROM $events`;
  const { data: totalEventsData, isLoading: totalEventsLoading } =
    useEventsQuery<TotalEventsResult>(totalEventsQuery);

  // Query for Error Rate
  const errorRateQuery = `
    SELECT 
      SUM(CASE WHEN type = 'Warning' THEN 1 ELSE 0 END) as warning_count,
      COUNT(*) as total_count
    FROM $events
  `;
  const { data: errorRateData, isLoading: errorRateLoading } =
    useEventsQuery<ErrorRateResult>(errorRateQuery);

  // Query for Failed Pods
  const failedPodsQuery = `
    SELECT COUNT(*) as failed_pods_count 
    FROM $events 
    WHERE reason IN ('FailedScheduling', 'Evicted', 'FailedCreatePodSandBox')
  `;
  const { data: failedPodsData, isLoading: failedPodsLoading } =
    useEventsQuery<FailedPodsResult>(failedPodsQuery);

  // Query for Restarts (BackOff events)
  const restartsQuery = `
    SELECT COUNT(*) as restarts_count 
    FROM $events 
    WHERE reason = 'BackOff'
  `;
  const { data: restartsData, isLoading: restartsLoading } =
    useEventsQuery<RestartsResult>(restartsQuery);

  // Query for Node Issues
  const nodeIssuesQuery = `
    SELECT COUNT(*) as node_issues_count 
    FROM $events 
    WHERE type = 'Warning' AND reason IN (
      'NodeNotReady', 'NodeHasDiskPressure', 'Unhealthy', 'TaintManagerEviction', 
      'NodeNotSchedulable', 'ImageGCFailed', 'FreeDiskSpaceFailed', 'FailedSync'
    )
  `;
  const { data: nodeIssuesData, isLoading: nodeIssuesLoading } =
    useEventsQuery<NodeIssuesResult>(nodeIssuesQuery);

  // Query for Network Events
  const networkEventsQuery = `
    SELECT COUNT(*) as network_events_count 
    FROM $events 
    WHERE type = 'Warning' AND reason IN (
      'FailedToCreateEndpoint', 'FailedToUpdateEndpoint', 'ErrUpdateFailed'
    )
  `;
  const { data: networkEventsData, isLoading: networkEventsLoading } =
    useEventsQuery<NetworkEventsResult>(networkEventsQuery);

  // Query for Storage Events
  const storageEventsQuery = `
    SELECT COUNT(*) as storage_events_count 
    FROM $events 
    WHERE type = 'Warning' AND reason IN (
      'FailedMount', 'VolumeFailedDelete', 'InvalidDiskCapacity'
    )
  `;
  const { data: storageEventsData, isLoading: storageEventsLoading } =
    useEventsQuery<StorageEventsResult>(storageEventsQuery);

  // Helper function to format numbers
  const formatNumber = (num: number): string => {
    if (num >= 1000000) {
      return `${(num / 1000000).toFixed(1)}M`;
    } else if (num >= 1000) {
      return `${(num / 1000).toFixed(1)}K`;
    }
    return num.toString();
  };

  // Helper function to calculate percentage
  const calculatePercentage = (part: number, total: number): string => {
    if (total === 0) return "0%";
    return `${((part / total) * 100).toFixed(1)}%`;
  };

  // Handle metric card clicks
  const handleMetricClick = (title: string) => {
    const queries: Record<string, string> = {
      "Total Events":
        "SELECT * FROM $events ORDER BY metadata.creationTimestamp DESC LIMIT 100",
      "Error Rate": "type='Warning'",
      "Failed Pods":
        "reason IN ('FailedScheduling', 'Evicted', 'FailedCreatePodSandBox')",
      Restarts: "reason='BackOff'",
      "New Warnings": "type='Warning'",
      "Node Issues":
        "type='Warning' AND reason IN ('NodeNotReady', 'NodeHasDiskPressure', 'Unhealthy', 'TaintManagerEviction', 'NodeNotSchedulable', 'ImageGCFailed', 'FreeDiskSpaceFailed', 'FailedSync')",
      "Network Events":
        "type='Warning' AND reason IN ('FailedToCreateEndpoint', 'FailedToUpdateEndpoint', 'ErrUpdateFailed')",
      "Storage Events":
        "type='Warning' AND reason IN ('FailedMount', 'VolumeFailedDelete', 'InvalidDiskCapacity')",
    };

    const query = queries[title];
    if (query) {
      if (title === "Total Events") {
        setQuery(query);
      } else {
        setWhereClause(query);
      }
    }
  };

  // Check if any queries are still loading
  const isLoading =
    totalEventsLoading ||
    errorRateLoading ||
    failedPodsLoading ||
    restartsLoading ||
    nodeIssuesLoading ||
    networkEventsLoading ||
    storageEventsLoading;

  if (isLoading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", py: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  // Prepare metrics data
  const totalEvents = totalEventsData?.[0]?.total_count || 0;
  const warningEvents = errorRateData?.[0]?.warning_count || 0;
  const totalForErrorRate = errorRateData?.[0]?.total_count || 0;
  const errorRate = calculatePercentage(warningEvents, totalForErrorRate);

  const metrics = [
    {
      title: "Total Events",
      value: formatNumber(totalEvents),
      subtitle: "All events in time range",
    },
    {
      title: "Error Rate",
      value: errorRate,
      subtitle: "Warning events ratio",
    },
    {
      title: "Failed Pods",
      value: formatNumber(failedPodsData?.[0]?.failed_pods_count || 0),
      subtitle: "Pod scheduling failures",
    },
    {
      title: "Restarts",
      value: formatNumber(restartsData?.[0]?.restarts_count || 0),
      subtitle: "Container restart failures",
    },
    {
      title: "New Warnings",
      value: formatNumber(warningEvents),
      subtitle: "Warning events",
    },
    {
      title: "Node Issues",
      value: formatNumber(nodeIssuesData?.[0]?.node_issues_count || 0),

      subtitle: "Node-related problems",
    },
    {
      title: "Network Events",
      value: formatNumber(networkEventsData?.[0]?.network_events_count || 0),
      subtitle: "Network-related issues",
    },
    {
      title: "Storage Events",
      value: formatNumber(storageEventsData?.[0]?.storage_events_count || 0),
      subtitle: "Storage-related issues",
    },
  ];

  return (
    <Grid container spacing={3}>
      {metrics.map((metric, index) => (
        <Grid size={{ xs: 12, sm: 6, md: 3 }} key={index}>
          <MetricCard
            title={metric.title}
            value={metric.value}
            subtitle={metric.subtitle}
            onClick={() => handleMetricClick(metric.title)}
          />
        </Grid>
      ))}
    </Grid>
  );
};

export default MetricsOverview;
