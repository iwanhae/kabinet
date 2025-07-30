import React, { useMemo } from "react";
import Chart from "react-apexcharts";
import { type ApexOptions } from "apexcharts";
import { Box, Typography, CircularProgress, Alert, Chip } from "@mui/material";
import { useEventsQuery } from "../hooks/useEventsQuery";
import { useTimeRangeStore } from "../stores/timeRangeStore";
import { getDynamicInterval } from "../utils/time";
import dayjs from "dayjs";
import { useLocation } from "wouter";

interface EventTimelineData {
  time_bucket: string;
  type: string;
  count: number;
}

const EventsTimelineChart: React.FC = () => {
  const { from, to } = useTimeRangeStore();
  const [, setLocation] = useLocation();

  const query = useMemo(() => {
    const interval = getDynamicInterval(from, to);

    return `
      SELECT 
        time_bucket(INTERVAL '${interval}', metadata.creationTimestamp) AS time_bucket,
        type,
        COUNT(*) AS count
      FROM $events 
      WHERE type IN ('Normal', 'Warning')
      AND metadata.creationTimestamp >= '${from}'
      AND metadata.creationTimestamp <= '${to}'
      GROUP BY time_bucket, type
      ORDER BY time_bucket, type
    `;
  }, [from, to]);

  const { data, error, isLoading } = useEventsQuery<EventTimelineData>(query);

  const chartData = useMemo(() => {
    if (!data) {
      return {
        series: [],
        categories: [],
        totalEvents: 0,
        dateRange: "",
        timeBuckets: [],
      };
    }

    const timeBuckets = [
      ...new Set(data.map((item) => item.time_bucket)),
    ].sort();
    const normalData = new Array(timeBuckets.length).fill(0);
    const warningData = new Array(timeBuckets.length).fill(0);

    let totalEvents = 0;
    data.forEach((item) => {
      const bucketIndex = timeBuckets.indexOf(item.time_bucket);
      if (bucketIndex !== -1) {
        totalEvents += item.count;
        if (item.type === "Warning") {
          warningData[bucketIndex] = item.count;
        } else if (item.type === "Normal") {
          normalData[bucketIndex] = item.count;
        }
      }
    });

    // Create more concise labels and reduce the number of displayed labels
    const categories = timeBuckets.map((bucket) => {
      const date = dayjs(bucket);
      const fromDate = dayjs(from);
      const toDate = dayjs(to);
      const durationHours = toDate.diff(fromDate, "hour");

      if (durationHours <= 24) {
        // For short periods, show time only
        return date.format("HH:mm");
      } else if (durationHours <= 24 * 7) {
        // For week periods, show month/day and time
        return date.format("MM/DD HH:mm");
      } else {
        // For longer periods, show date only
        return date.format("MM/DD");
      }
    });

    // Calculate date range for display
    const startDate =
      timeBuckets.length > 0 ? dayjs(timeBuckets[0]) : dayjs(from);
    const endDate =
      timeBuckets.length > 0
        ? dayjs(timeBuckets[timeBuckets.length - 1])
        : dayjs(to);
    const dateRange = `${startDate.format("MMM DD")} - ${endDate.format(
      "MMM DD, YYYY",
    )}`;

    return {
      series: [
        {
          name: "Warning Events",
          data: warningData,
        },
        {
          name: "Normal Events",
          data: normalData,
        },
      ],
      categories,
      totalEvents,
      dateRange,
      timeBuckets,
    };
  }, [data, from, to]);

  const handleDataPointClick = (
    _event: unknown,
    _chartContext: unknown,
    { dataPointIndex }: { dataPointIndex: number },
  ) => {
    if (dataPointIndex >= 0 && dataPointIndex < chartData.timeBuckets.length) {
      const selectedBucket = chartData.timeBuckets[dataPointIndex];
      const bucketStart = dayjs(selectedBucket);

      // Calculate the interval duration to get the end time
      const interval = getDynamicInterval(from, to);
      let bucketEnd = bucketStart;

      if (interval.includes("second")) {
        const seconds = parseInt(interval.match(/\d+/)?.[0] || "1");
        bucketEnd = bucketStart.add(seconds, "second");
      } else if (interval.includes("minute")) {
        const minutes = parseInt(interval.match(/\d+/)?.[0] || "1");
        bucketEnd = bucketStart.add(minutes, "minute");
      } else if (interval.includes("hour")) {
        const hours = parseInt(interval.match(/\d+/)?.[0] || "1");
        bucketEnd = bucketStart.add(hours, "hour");
      } else if (interval.includes("day")) {
        const days = parseInt(interval.match(/\d+/)?.[0] || "1");
        bucketEnd = bucketStart.add(days, "day");
      }

      const query = `metadata.creationTimestamp >= '${bucketStart.toISOString()}' 
AND metadata.creationTimestamp < '${bucketEnd.toISOString()}'`;

      setLocation(`/discover?where=${encodeURIComponent(query)}`);
    }
  };

  const chartOptions: ApexOptions = {
    chart: {
      type: "bar",
      height: 350,
      stacked: true,
      toolbar: {
        show: false,
      },
      zoom: {
        enabled: false,
      },
      animations: {
        enabled: false,
      },
      events: {
        dataPointSelection: handleDataPointClick,
      },
    },
    plotOptions: {
      bar: {
        columnWidth: "85%",
      },
    },
    colors: ["#f44336", "#2E93fA"], // Red for warnings, blue for normal
    xaxis: {
      categories: chartData.categories,
      labels: {
        rotate: -45,
        style: {
          fontSize: "11px",
        },
        maxHeight: 80,
        // Show fewer labels to prevent overlap
        showDuplicates: false,
      },
      tickAmount: Math.min(chartData.categories.length, 15), // Limit to max 15 ticks
    },
    yaxis: {
      title: {
        text: "Event Count",
      },
      labels: {
        formatter: (value) => {
          if (value >= 1000) {
            return (value / 1000).toFixed(1) + "K";
          }
          return value.toString();
        },
      },
    },
    legend: {
      position: "top",
      horizontalAlign: "right",
    },
    grid: {
      borderColor: "#e0e0e0",
      strokeDashArray: 3,
    },
    dataLabels: {
      enabled: false,
    },
    tooltip: {
      shared: true,
      intersect: false,
      x: {
        formatter: (
          _value: unknown,
          { dataPointIndex }: { dataPointIndex: number },
        ) => {
          const bucket = chartData.timeBuckets[dataPointIndex];
          const date = dayjs(bucket);
          return `Time: ${date.format("MMM DD, HH:mm")}`;
        },
      },
      y: {
        formatter: (
          value: number,
          { seriesIndex }: { seriesIndex: number },
        ) => {
          const type = seriesIndex === 0 ? "Warning" : "Normal";
          return `${value.toLocaleString()} ${type} events`;
        },
      },
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      custom: ({ series, dataPointIndex }: any) => {
        const warningCount = series[0][dataPointIndex] || 0;
        const normalCount = series[1][dataPointIndex] || 0;
        const total = warningCount + normalCount;
        const warningPct =
          total > 0 ? ((warningCount / total) * 100).toFixed(1) : "0";
        const normalPct =
          total > 0 ? ((normalCount / total) * 100).toFixed(1) : "0";

        const bucket = chartData.timeBuckets[dataPointIndex];
        const date = dayjs(bucket);

        return `
          <div style="padding: 12px; background: white; border-radius: 6px; box-shadow: 0 4px 12px rgba(0,0,0,0.15);">
            <div style="font-weight: 600; margin-bottom: 8px; color: #333;">
              ${date.format("MMM DD, HH:mm")}
            </div>
            <div style="margin-bottom: 4px;">
              <span style="color: #f44336;">●</span> Warning: <strong>${warningCount.toLocaleString()}</strong> (${warningPct}%)
            </div>
            <div style="margin-bottom: 4px;">
              <span style="color: #2E93fA;">●</span> Normal: <strong>${normalCount.toLocaleString()}</strong> (${normalPct}%)
            </div>
            <div style="border-top: 1px solid #eee; padding-top: 4px; margin-top: 8px; font-weight: 600;">
              Total: ${total.toLocaleString()} events
            </div>
            <div style="margin-top: 8px; font-size: 11px; color: #666;">
              Click to view details →
            </div>
          </div>
        `;
      },
    },
    responsive: [
      {
        breakpoint: 768,
        options: {
          chart: {
            height: 300,
          },
          xaxis: {
            labels: {
              rotate: -90,
              style: {
                fontSize: "10px",
              },
            },
            tickAmount: Math.min(chartData.categories.length, 10),
          },
        },
      },
      {
        breakpoint: 480,
        options: {
          chart: {
            height: 280,
          },
          xaxis: {
            labels: {
              rotate: -90,
              style: {
                fontSize: "9px",
              },
            },
            tickAmount: Math.min(chartData.categories.length, 8),
          },
          legend: {
            position: "bottom",
          },
        },
      },
    ],
  };

  if (isLoading) {
    return (
      <Box
        sx={{
          height: 350,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ mb: 2 }}>
        Failed to load timeline data: {error.message}
      </Alert>
    );
  }

  if (!data || data.length === 0) {
    return (
      <Box
        sx={{
          height: 350,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          flexDirection: "column",
          color: "text.secondary",
        }}
      >
        <Typography variant="body1">
          No data available for the selected time range
        </Typography>
      </Box>
    );
  }

  return (
    <Box>
      {/* Dynamic Header with actual data */}
      <Box
        sx={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          mb: 3,
        }}
      >
        <Box>
          <Typography variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
            Events Timeline
          </Typography>
          <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
            <Typography variant="body2" color="text.secondary">
              {chartData.dateRange}
            </Typography>
            <Chip
              label={`Events: ${chartData.totalEvents.toLocaleString()}`}
              size="small"
              sx={{
                backgroundColor: "primary.main",
                color: "white",
                fontWeight: 600,
              }}
            />
          </Box>
        </Box>
      </Box>

      <Chart
        options={chartOptions}
        series={chartData.series}
        type="bar"
        height={350}
      />
    </Box>
  );
};

export default EventsTimelineChart;
