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

interface NamespaceData {
  namespace: string;
  count: number;
}

interface TopNoisyNamespacesProps {
  data?: NamespaceData[];
}

const TopNoisyNamespaces: React.FC<TopNoisyNamespacesProps> = ({ data }) => {
  const query = `
    SELECT metadata.namespace as namespace, COUNT(*) as count 
    FROM $events 
    WHERE type = 'Warning' AND metadata.namespace IS NOT NULL 
    GROUP BY metadata.namespace 
    ORDER BY count DESC;`;

  const {
    data: queryData,
    error,
    isLoading,
  } = useEventsQuery<NamespaceData>(query);

  const namespaces = data || queryData || [];

  const formatCount = (count: number): string => {
    if (count >= 1000) {
      return `${(count / 1000).toFixed(1)}K`;
    }
    return count.toString();
  };

  const getCountColor = (count: number, maxCount: number) => {
    const ratio = count / maxCount;
    if (ratio > 0.8) return "error";
    if (ratio > 0.5) return "warning";
    return "info";
  };

  const maxCount = Math.max(...namespaces.map((ns) => ns.count));

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
            Failed to load namespace data
          </Alert>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardContent sx={{ p: 3 }}>
        <Typography variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
          Top Noisy Namespaces
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Namespaces with most warning events
        </Typography>

        <List dense sx={{ p: 0, maxHeight: 500, overflowY: "auto" }}>
          {namespaces.map((item, index) => {
            const query = `metadata.namespace='${item.namespace}' AND type='Warning'`;
            const href = `/discover?where=${encodeURIComponent(query)}`;

            return (
              <ListItem
                key={item.namespace}
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
                            overflow: "hidden",
                            textOverflow: "ellipsis",
                            whiteSpace: "nowrap",
                          }}
                        >
                          {item.namespace}
                        </Typography>
                      </Link>
                    }
                    sx={{ m: 0, flex: 1 }}
                  />

                  <Chip
                    label={formatCount(item.count)}
                    size="small"
                    color={getCountColor(item.count, maxCount)}
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

export default TopNoisyNamespaces;
