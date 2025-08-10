import React from "react";
import {
  Typography,
  Box,
  TextField,
  Button,
  CircularProgress,
  alpha,
} from "@mui/material";
import { styled } from "@mui/material/styles";
import { useTimeRange } from "../hooks/useUrlParams";

interface QueryFormProps {
  whereClause: string;
  onWhereClauseChange: (value: string) => void;
  onExecuteQuery: () => void;
  isLoading: boolean;
}

const QueryBox = styled(Box)(({ theme }) => ({
  display: "flex",
  gap: theme.spacing(2),
  marginBottom: theme.spacing(3),
  alignItems: "flex-start",
}));

const QueryForm: React.FC<QueryFormProps> = ({
  whereClause,
  onWhereClauseChange,
  onExecuteQuery,
  isLoading,
}) => {
  const { from, to } = useTimeRange();

  const handleKeyPress = (event: React.KeyboardEvent) => {
    if (event.key === "Enter" && event.ctrlKey) {
      onExecuteQuery();
    }
  };

  const handleDownload = () => {
    const trimmedWhere = whereClause.trim() || "1=1";
    const params = new URLSearchParams({
      where: trimmedWhere,
      from,
      to,
    });
    window.location.href = `/download?${params.toString()}`;
  };

  return (
    <Box mb={4}>
      <Typography variant="h4" gutterBottom sx={{ fontWeight: 700 }}>
        Event Discovery
      </Typography>
      <Typography
        variant="body1"
        color="text.secondary"
        gutterBottom
        sx={{ fontSize: "1.1rem", mb: 3 }}
      >
        Enter a WHERE clause to filter Kubernetes events. Use fields like{" "}
        <code
          style={{
            backgroundColor: "#f5f5f5",
            padding: "2px 6px",
            borderRadius: "4px",
          }}
        >
          type = &apos;Warning&apos;
        </code>
        ,{" "}
        <code
          style={{
            backgroundColor: "#f5f5f5",
            padding: "2px 6px",
            borderRadius: "4px",
          }}
        >
          metadata.namespace = &apos;kube-system&apos;
        </code>
        , or{" "}
        <code
          style={{
            backgroundColor: "#f5f5f5",
            padding: "2px 6px",
            borderRadius: "4px",
          }}
        >
          reason = &apos;FailedMount&apos;
        </code>
        .
      </Typography>

      <QueryBox>
        <TextField
          fullWidth
          multiline
          rows={3}
          label="WHERE clause"
          placeholder="type = 'Warning' AND metadata.namespace = 'kube-system'"
          value={whereClause}
          onChange={(e) => onWhereClauseChange(e.target.value)}
          onKeyPress={handleKeyPress}
          helperText="Press Ctrl+Enter to execute query"
          variant="outlined"
          sx={{
            "& .MuiOutlinedInput-root": {
              backgroundColor: alpha("#f8f9fa", 0.6),
              borderRadius: 1.5,
              "&:hover": {
                backgroundColor: alpha("#f5f5f5", 0.9),
              },
              "&.Mui-focused": {
                backgroundColor: "white",
              },
            },
          }}
        />
        <Button
          variant="contained"
          onClick={onExecuteQuery}
          size="large"
          sx={{
            minWidth: 120,
            height: "fit-content",
            px: 3,
            py: 1.5,
            borderRadius: 1.5,
            fontWeight: 600,
            boxShadow: 1,
            "&:hover": {
              boxShadow: 3,
            },
          }}
        >
          {isLoading ? (
            <CircularProgress size={20} color="inherit" />
          ) : (
            "Execute"
          )}
        </Button>
        <Button
          variant="outlined"
          onClick={handleDownload}
          size="large"
          sx={{
            minWidth: 120,
            height: "fit-content",
            px: 3,
            py: 1.5,
            borderRadius: 1.5,
            fontWeight: 600,
          }}
        >
          Download
        </Button>
      </QueryBox>
    </Box>
  );
};

export default QueryForm;
