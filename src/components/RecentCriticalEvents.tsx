import React from "react";
import {
  Typography,
  Box,
  Card,
  CardContent,
  List,
  ListItem,
  Chip,
  alpha,
  CircularProgress,
  Alert,
} from "@mui/material";
import { useEventsQuery } from "../hooks/useEventsQuery";
import { Link } from "./Link";

interface CriticalEvent {
  uid: string;
  type: string;
  reason: string;
  namespace: string;
  objectName: string;
  message: string;
  creationTimestamp: string;
}

interface RecentCriticalEventsProps {
  data?: CriticalEvent[];
}

const RecentCriticalEvents: React.FC<RecentCriticalEventsProps> = ({
  data,
}) => {
  const query = `
    SELECT 
      metadata.uid as uid,
      type,
      reason,
      metadata.namespace as namespace,
      involvedObject.name as objectName,
      message,
      lastTimestamp as creationTimestamp
    FROM $events 
    WHERE type = 'Warning' 
    ORDER BY lastTimestamp DESC 
    LIMIT 20
  `;

  const {
    data: queryData,
    error,
    isLoading,
  } = useEventsQuery<CriticalEvent>(query);

  const events = data || queryData || [];

  const formatTimestamp = (timestamp: string): string => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMinutes = Math.floor(
      (now.getTime() - date.getTime()) / (1000 * 60),
    );

    if (diffMinutes < 1) return "just now";
    if (diffMinutes < 60) return `${diffMinutes}m ago`;
    if (diffMinutes < 1440) return `${Math.floor(diffMinutes / 60)}h ago`;
    return date.toLocaleDateString();
  };

  const getEventTypeColor = (type: string) => {
    return type === "Warning" ? "warning" : "error";
  };

  if (isLoading) {
    return (
      <Card>
        <CardContent sx={{ p: 3, display: "flex", justifyContent: "center" }}>
          <CircularProgress size={24} />
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardContent sx={{ p: 3 }}>
          <Alert severity="error" sx={{ fontSize: "0.875rem" }}>
            Failed to load recent events data
          </Alert>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardContent sx={{ p: 3 }}>
        <Typography variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
          Recent Critical Events
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Latest warning events across the cluster
        </Typography>

        <List dense sx={{ p: 0, maxHeight: 500, overflowY: "auto" }}>
          {events.map((event) => {
            const query = `reason='${event.reason}' AND metadata.namespace='${event.namespace}'`;

            return (
              <ListItem
                key={event.uid}
                sx={{
                  px: 0,
                  py: 1.5,
                  borderBottom: `1px solid ${alpha("#e0e0e0", 0.3)}`,
                  "&:last-child": { borderBottom: "none" },
                  "&:hover": {
                    backgroundColor: alpha("#f5f5f5", 0.5),
                    borderRadius: 1,
                  },
                }}
              >
                <Box sx={{ width: "100%" }}>
                  <Box sx={{ display: "flex", alignItems: "center", mb: 0.5 }}>
                    <Chip
                      label={event.type}
                      size="small"
                      color={getEventTypeColor(event.type)}
                      sx={{ mr: 1, minWidth: 70, fontSize: "0.7rem" }}
                    />
                    <Link page="discover" params={{ where: query }}>
                      <Typography
                        variant="body2"
                        sx={{
                          fontWeight: 600,
                          color: "primary.main",
                          "&:hover": { textDecoration: "underline" },
                        }}
                      >
                        {event.reason}
                      </Typography>
                    </Link>
                    <Typography
                      variant="caption"
                      color="text.secondary"
                      sx={{ ml: "auto", fontSize: "0.7rem" }}
                    >
                      {formatTimestamp(event.creationTimestamp)}
                    </Typography>
                  </Box>

                  <Box sx={{ display: "flex", alignItems: "center", mb: 0.5 }}>
                    <Typography
                      variant="caption"
                      color="text.secondary"
                      sx={{ mr: 1 }}
                    >
                      {event.namespace}
                    </Typography>
                    <Typography variant="caption" sx={{ fontWeight: 500 }}>
                      {event.objectName}
                    </Typography>
                  </Box>

                  <Typography
                    variant="body2"
                    color="text.secondary"
                    sx={{
                      fontSize: "0.75rem",
                      lineHeight: 1.3,
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                      whiteSpace: "nowrap",
                    }}
                  >
                    {event.message}
                  </Typography>
                </Box>
              </ListItem>
            );
          })}
        </List>
      </CardContent>
    </Card>
  );
};

export default RecentCriticalEvents;
