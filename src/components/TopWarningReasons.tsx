import React from "react";
import {
  Typography,
  Box,
  Card,
  CardContent,
  List,
  ListItem,
  ListItemText,
  Chip,
  alpha,
  CircularProgress,
  Alert,
} from "@mui/material";
import { Link } from "wouter";
import { useEventsQuery } from "../hooks/useEventsQuery";

interface WarningReasonData {
  reason: string;
  count: number;
}

interface TopWarningReasonsProps {
  data?: WarningReasonData[];
}

const TopWarningReasons: React.FC<TopWarningReasonsProps> = ({ data }) => {
  const query = `
    SELECT reason, COUNT(*) as count 
    FROM $events 
    WHERE type = 'Warning' 
    GROUP BY reason 
    ORDER BY count DESC;`;

  const {
    data: queryData,
    error,
    isLoading,
  } = useEventsQuery<WarningReasonData>(query);

  const reasons = data || queryData || [];

  const formatCount = (count: number): string => {
    if (count >= 1000) {
      return `${(count / 1000).toFixed(1)}K`;
    }
    return count.toString();
  };

  const getReasonColor = (reason: string) => {
    if (reason.includes("Failed") || reason.includes("Error")) return "error";
    if (
      reason.includes("Unhealthy") ||
      reason.includes("Warning") ||
      reason.includes("OutOf")
    )
      return "warning";
    return "info";
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
            Failed to load warning reasons data
          </Alert>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardContent sx={{ p: 3 }}>
        <Typography variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
          Top Warning Reasons
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Most frequent warning event types
        </Typography>

        <List dense sx={{ p: 0, maxHeight: 500, overflowY: "auto" }}>
          {reasons.map((item, index) => {
            const query = `reason='${item.reason}' AND type='Warning'`;
            const href = `/discover?where=${encodeURIComponent(query)}`;

            return (
              <ListItem
                key={item.reason}
                sx={{
                  px: 0,
                  py: 1,
                  "&:hover": {
                    backgroundColor: alpha("#f5f5f5", 0.5),
                    borderRadius: 1,
                  },
                }}
              >
                <Box
                  sx={{ display: "flex", alignItems: "center", width: "100%" }}
                >
                  <Box
                    sx={{
                      minWidth: 24,
                      height: 24,
                      borderRadius: "50%",
                      backgroundColor: "primary.main",
                      color: "white",
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                      fontSize: "0.75rem",
                      fontWeight: 600,
                      mr: 2,
                    }}
                  >
                    {index + 1}
                  </Box>

                  <ListItemText
                    primary={
                      <Link href={href} style={{ textDecoration: "none" }}>
                        <Typography
                          variant="body2"
                          sx={{
                            fontWeight: 500,
                            color: "primary.main",
                            "&:hover": { textDecoration: "underline" },
                          }}
                        >
                          {item.reason}
                        </Typography>
                      </Link>
                    }
                    sx={{ m: 0, flex: 1 }}
                  />

                  <Chip
                    label={formatCount(item.count)}
                    size="small"
                    color={getReasonColor(item.reason)}
                    sx={{
                      minWidth: 48,
                      fontWeight: 600,
                      fontSize: "0.75rem",
                    }}
                  />
                </Box>
              </ListItem>
            );
          })}
        </List>
      </CardContent>
    </Card>
  );
};

export default TopWarningReasons;
