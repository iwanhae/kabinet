import React, { useState } from "react";
import {
  Typography,
  Box,
  TextField,
  Button,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableRow,
  TableHead,
  Chip,
  Alert,
  CircularProgress,
  Drawer,
  IconButton,
  Divider,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  styled,
  alpha,
} from "@mui/material";
import { ExpandMore, Close } from "@mui/icons-material";
import { useEventsQuery } from "../hooks/useEventsQuery";

// 이벤트의 전체 구조를 정의하는 타입
interface EventResult {
  kind: string;
  apiVersion: string;
  metadata: {
    name: string;
    namespace: string;
    uid: string;
    resourceVersion: string;
    creationTimestamp: string;
  };
  involvedObject: {
    kind: string;
    namespace: string;
    name: string;
    uid: string;
    apiVersion: string;
    resourceVersion: string;
    fieldPath?: string;
  };
  reason: string;
  message: string;
  source: {
    component: string;
    host: string;
  };
  firstTimestamp: string;
  lastTimestamp: string;
  count: number;
  type: string;
  eventTime?: string;
  series?: {
    count: number;
    lastObservedTime: string;
  };
  action?: string;
  related?: {
    kind: string;
    namespace: string;
    name: string;
    uid: string;
    apiVersion: string;
    resourceVersion: string;
    fieldPath?: string;
  };
  reportingComponent?: string;
  reportingInstance?: string;
}

const StyledTableRow = styled(TableRow)<{ eventType: string }>(
  ({ theme, eventType }) => ({
    cursor: "pointer",
    borderLeft: `2px solid ${
      eventType === "Warning"
        ? theme.palette.warning.main
        : theme.palette.success.main
    }`,
    "&:hover": {
      backgroundColor: alpha(theme.palette.primary.main, 0.04),
    },
    transition: "all 0.2s ease-in-out",
  }),
);

const QueryBox = styled(Box)(({ theme }) => ({
  display: "flex",
  gap: theme.spacing(2),
  marginBottom: theme.spacing(3),
  alignItems: "flex-start",
}));

const EventTypeChip = styled(Chip)<{ eventType: string }>(
  ({ theme, eventType }) => ({
    backgroundColor:
      eventType === "Warning"
        ? alpha(theme.palette.warning.main, 0.15)
        : alpha(theme.palette.success.main, 0.15),
    color:
      eventType === "Warning"
        ? theme.palette.warning.dark
        : theme.palette.success.dark,
    fontWeight: 600,
    border: `1px solid ${
      eventType === "Warning"
        ? alpha(theme.palette.warning.main, 0.3)
        : alpha(theme.palette.success.main, 0.3)
    }`,
    minWidth: 70,
  }),
);

const DrawerHeader = styled(Box)(({ theme }) => ({
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  padding: theme.spacing(2, 3),
  borderBottom: `1px solid ${theme.palette.divider}`,
}));

const DrawerContent = styled(Box)(({ theme }) => ({
  padding: theme.spacing(0),
  width: 500,
}));

const MessageBox = styled(Box)(({ theme }) => ({
  backgroundColor: alpha(theme.palette.grey[100], 0.5),
  padding: theme.spacing(1.5, 2),
  borderRadius: theme.spacing(1),
  border: `1px solid ${alpha(theme.palette.divider, 0.1)}`,
  marginBottom: theme.spacing(2),
}));

const TimeStampChip = styled(Chip)(({ theme }) => ({
  backgroundColor: alpha(theme.palette.info.main, 0.08),
  color: theme.palette.info.dark,
  fontSize: "0.75rem",
  height: "24px",
  marginRight: theme.spacing(1),
  marginBottom: theme.spacing(0.5),
}));

const Discover: React.FC = () => {
  const [whereClause, setWhereClause] = useState("");
  const [executedQuery, setExecutedQuery] = useState<string | null>(null);
  const [selectedEvent, setSelectedEvent] = useState<EventResult | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);

  // 쿼리가 실행된 경우에만 데이터를 가져옴
  const query = executedQuery
    ? `SELECT * FROM $events WHERE ${executedQuery} ORDER BY metadata.creationTimestamp DESC LIMIT 100`
    : null;

  const { data, error, isLoading } = useEventsQuery<EventResult>(query);

  const handleExecuteQuery = () => {
    if (whereClause.trim()) {
      setExecutedQuery(whereClause.trim());
    }
  };

  const handleKeyPress = (event: React.KeyboardEvent) => {
    if (event.key === "Enter" && event.ctrlKey) {
      handleExecuteQuery();
    }
  };

  const handleRowClick = (event: EventResult) => {
    setSelectedEvent(event);
    setDrawerOpen(true);
  };

  const handleDrawerClose = () => {
    setDrawerOpen(false);
    setSelectedEvent(null);
  };

  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  return (
    <Box>
      <Box mb={4}>
        <Typography variant="h4" gutterBottom sx={{ fontWeight: 700 }}>
          Event Discovery
        </Typography>
        <Typography
          variant="body1"
          color="text.secondary"
          gutterBottom
          sx={{ fontSize: "1.1rem" }}
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
      </Box>

      <QueryBox>
        <TextField
          fullWidth
          multiline
          rows={3}
          label="WHERE clause"
          placeholder="type = 'Warning' AND metadata.namespace = 'kube-system'"
          value={whereClause}
          onChange={(e) => setWhereClause(e.target.value)}
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
          onClick={handleExecuteQuery}
          disabled={!whereClause.trim() || isLoading}
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
      </QueryBox>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          Query Error: {error.message}
        </Alert>
      )}

      {executedQuery && (
        <Typography variant="body2" color="text.secondary" gutterBottom>
          Query:{" "}
          <code>
            SELECT * FROM $events WHERE {executedQuery} ORDER BY
            metadata.creationTimestamp DESC LIMIT 100
          </code>
        </Typography>
      )}

      {isLoading && (
        <Box display="flex" justifyContent="center" p={4}>
          <CircularProgress />
        </Box>
      )}

      {data && data.length === 0 && (
        <Alert severity="info">No events found matching your criteria.</Alert>
      )}

      {data && data.length > 0 && (
        <Box>
          <Box
            sx={{
              display: "flex",
              alignItems: "center",
              gap: 2,
              mb: 2,
              p: 2,
              backgroundColor: alpha("#f5f5f5", 0.7),
              borderRadius: 1.5,
              border: `1px solid ${alpha("#e0e0e0", 0.5)}`,
            }}
          >
            <Typography
              variant="subtitle1"
              sx={{ fontWeight: 600, color: "text.primary" }}
            >
              Found {data.length} events
            </Typography>
          </Box>

          <TableContainer
            component={Paper}
            elevation={1}
            sx={{
              borderRadius: 1.5,
              border: `1px solid ${alpha("#e0e0e0", 0.3)}`,
              overflow: "hidden",
            }}
          >
            <Table sx={{ tableLayout: "fixed", width: "100%" }}>
              <TableHead>
                <TableRow sx={{ backgroundColor: alpha("#f8f9fa", 0.8) }}>
                  <TableCell
                    sx={{
                      fontWeight: 600,
                      fontSize: "0.875rem",
                      width: "90px",
                      minWidth: "90px",
                    }}
                  >
                    Type
                  </TableCell>
                  <TableCell
                    sx={{
                      fontWeight: 600,
                      fontSize: "0.875rem",
                      width: "150px",
                      minWidth: "150px",
                    }}
                  >
                    Created
                  </TableCell>
                  <TableCell
                    sx={{
                      fontWeight: 600,
                      fontSize: "0.875rem",
                      width: "120px",
                      minWidth: "120px",
                    }}
                  >
                    Kind
                  </TableCell>
                  <TableCell
                    sx={{
                      fontWeight: 600,
                      fontSize: "0.875rem",
                      width: "200px",
                      minWidth: "200px",
                    }}
                  >
                    Namespace / Name
                  </TableCell>
                  <TableCell
                    sx={{
                      fontWeight: 600,
                      fontSize: "0.875rem",
                      width: "180px",
                      minWidth: "180px",
                    }}
                  >
                    Reason
                  </TableCell>
                  <TableCell
                    sx={{
                      fontWeight: 600,
                      fontSize: "0.875rem",
                      width: "auto",
                    }}
                  >
                    Message
                  </TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {data.map((event, index) => (
                  <StyledTableRow
                    key={event.metadata.uid || index}
                    eventType={event.type}
                    onClick={() => handleRowClick(event)}
                  >
                    <TableCell
                      sx={{ py: 1.5, width: "90px", minWidth: "90px" }}
                    >
                      <EventTypeChip
                        label={event.type}
                        size="small"
                        eventType={event.type}
                      />
                    </TableCell>
                    <TableCell
                      sx={{ py: 1.5, width: "150px", minWidth: "150px" }}
                    >
                      <Typography
                        variant="body2"
                        color="text.secondary"
                        sx={{ fontSize: "0.8rem" }}
                      >
                        {formatTimestamp(event.metadata.creationTimestamp)}
                      </Typography>
                    </TableCell>
                    <TableCell
                      sx={{ py: 1.5, width: "120px", minWidth: "120px" }}
                    >
                      <Typography
                        variant="body2"
                        color="text.secondary"
                        sx={{
                          fontWeight: 600,
                          fontSize: "0.85rem",
                          overflow: "hidden",
                          textOverflow: "ellipsis",
                          whiteSpace: "nowrap",
                        }}
                      >
                        {event.involvedObject.kind}
                      </Typography>
                    </TableCell>
                    <TableCell
                      sx={{ py: 1.5, width: "200px", minWidth: "200px" }}
                    >
                      <Typography
                        variant="body2"
                        sx={{
                          fontSize: "0.85rem",
                          overflow: "hidden",
                          textOverflow: "ellipsis",
                          whiteSpace: "nowrap",
                        }}
                      >
                        {event.metadata.namespace
                          ? event.metadata.namespace + " /"
                          : ""}
                        <br />
                        {event.involvedObject.name}
                      </Typography>
                    </TableCell>
                    <TableCell
                      sx={{
                        fontWeight: 600,
                        py: 1.5,
                        fontSize: "0.9rem",
                        width: "180px",
                        minWidth: "180px",
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                        whiteSpace: "nowrap",
                      }}
                    >
                      {event.reason}
                    </TableCell>
                    <TableCell sx={{ py: 1.5 }}>
                      <Typography
                        variant="body2"
                        sx={{
                          overflow: "hidden",
                          textOverflow: "ellipsis",
                          whiteSpace: "nowrap",
                          fontSize: "0.85rem",
                        }}
                      >
                        {event.message}
                      </Typography>
                    </TableCell>
                  </StyledTableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Box>
      )}

      {/* Event Detail Drawer */}
      <Drawer
        anchor="right"
        open={drawerOpen}
        onClose={handleDrawerClose}
        variant="temporary"
      >
        <DrawerContent>
          <DrawerHeader>
            <Typography variant="h6" sx={{ fontWeight: 600 }}>
              Event Details
            </Typography>
            <IconButton onClick={handleDrawerClose} size="small">
              <Close />
            </IconButton>
          </DrawerHeader>

          {selectedEvent && (
            <Box sx={{ p: 3 }}>
              {/* Event Header */}
              <Box sx={{ mb: 3 }}>
                <Box
                  sx={{ display: "flex", alignItems: "center", gap: 2, mb: 2 }}
                >
                  <EventTypeChip
                    label={selectedEvent.type}
                    size="medium"
                    eventType={selectedEvent.type}
                  />
                  <Typography variant="h5" sx={{ fontWeight: 600 }}>
                    {selectedEvent.reason}
                  </Typography>
                </Box>
                <Typography variant="body1" color="text.secondary">
                  {selectedEvent.involvedObject.kind}/
                  {selectedEvent.involvedObject.name}
                  {selectedEvent.involvedObject.namespace &&
                    ` in ${selectedEvent.involvedObject.namespace}`}
                </Typography>
              </Box>

              {/* Message */}
              <MessageBox>
                <Typography variant="h6" sx={{ mb: 1, fontWeight: 600 }}>
                  Message
                </Typography>
                <Typography variant="body1" sx={{ lineHeight: 1.6 }}>
                  {selectedEvent.message}
                </Typography>
              </MessageBox>

              {/* Timestamps */}
              <Box sx={{ mb: 3 }}>
                <Typography variant="h6" sx={{ mb: 2, fontWeight: 600 }}>
                  Timestamps
                </Typography>
                <Box sx={{ display: "flex", flexWrap: "wrap", gap: 1 }}>
                  <TimeStampChip
                    label={`Created: ${formatTimestamp(selectedEvent.metadata.creationTimestamp)}`}
                    size="small"
                  />
                  <TimeStampChip
                    label={`First: ${formatTimestamp(selectedEvent.firstTimestamp)}`}
                    size="small"
                  />
                  <TimeStampChip
                    label={`Last: ${formatTimestamp(selectedEvent.lastTimestamp)}`}
                    size="small"
                  />
                  <TimeStampChip
                    label={`Count: ${selectedEvent.count}`}
                    size="small"
                  />
                </Box>
              </Box>

              <Divider sx={{ my: 3 }} />

              {/* Detailed Information */}
              <Accordion defaultExpanded>
                <AccordionSummary expandIcon={<ExpandMore />}>
                  <Typography variant="h6" sx={{ fontWeight: 600 }}>
                    Detailed Information
                  </Typography>
                </AccordionSummary>
                <AccordionDetails>
                  <TableContainer>
                    <Table size="small">
                      <TableBody>
                        <TableRow>
                          <TableCell
                            component="th"
                            scope="row"
                            sx={{ fontWeight: 600, width: "40%" }}
                          >
                            UID
                          </TableCell>
                          <TableCell
                            sx={{
                              fontFamily: "monospace",
                              fontSize: "0.875rem",
                            }}
                          >
                            {selectedEvent.metadata.uid}
                          </TableCell>
                        </TableRow>
                        <TableRow>
                          <TableCell
                            component="th"
                            scope="row"
                            sx={{ fontWeight: 600 }}
                          >
                            Namespace
                          </TableCell>
                          <TableCell>
                            {selectedEvent.metadata.namespace || "default"}
                          </TableCell>
                        </TableRow>
                        <TableRow>
                          <TableCell
                            component="th"
                            scope="row"
                            sx={{ fontWeight: 600 }}
                          >
                            Source Component
                          </TableCell>
                          <TableCell>
                            {selectedEvent.source.component}
                          </TableCell>
                        </TableRow>
                        <TableRow>
                          <TableCell
                            component="th"
                            scope="row"
                            sx={{ fontWeight: 600 }}
                          >
                            Source Host
                          </TableCell>
                          <TableCell>{selectedEvent.source.host}</TableCell>
                        </TableRow>
                        <TableRow>
                          <TableCell
                            component="th"
                            scope="row"
                            sx={{ fontWeight: 600 }}
                          >
                            API Version
                          </TableCell>
                          <TableCell>{selectedEvent.apiVersion}</TableCell>
                        </TableRow>
                        <TableRow>
                          <TableCell
                            component="th"
                            scope="row"
                            sx={{ fontWeight: 600 }}
                          >
                            Kind
                          </TableCell>
                          <TableCell>{selectedEvent.kind}</TableCell>
                        </TableRow>
                        {selectedEvent.action && (
                          <TableRow>
                            <TableCell
                              component="th"
                              scope="row"
                              sx={{ fontWeight: 600 }}
                            >
                              Action
                            </TableCell>
                            <TableCell>{selectedEvent.action}</TableCell>
                          </TableRow>
                        )}
                        {selectedEvent.reportingComponent && (
                          <TableRow>
                            <TableCell
                              component="th"
                              scope="row"
                              sx={{ fontWeight: 600 }}
                            >
                              Reporting Component
                            </TableCell>
                            <TableCell>
                              {selectedEvent.reportingComponent}
                            </TableCell>
                          </TableRow>
                        )}
                        {selectedEvent.reportingInstance && (
                          <TableRow>
                            <TableCell
                              component="th"
                              scope="row"
                              sx={{ fontWeight: 600 }}
                            >
                              Reporting Instance
                            </TableCell>
                            <TableCell>
                              {selectedEvent.reportingInstance}
                            </TableCell>
                          </TableRow>
                        )}
                        {selectedEvent.involvedObject.fieldPath && (
                          <TableRow>
                            <TableCell
                              component="th"
                              scope="row"
                              sx={{ fontWeight: 600 }}
                            >
                              Field Path
                            </TableCell>
                            <TableCell
                              sx={{
                                fontFamily: "monospace",
                                fontSize: "0.875rem",
                              }}
                            >
                              {selectedEvent.involvedObject.fieldPath}
                            </TableCell>
                          </TableRow>
                        )}
                        {selectedEvent.series && (
                          <TableRow>
                            <TableCell
                              component="th"
                              scope="row"
                              sx={{ fontWeight: 600 }}
                            >
                              Series Count
                            </TableCell>
                            <TableCell>{selectedEvent.series.count}</TableCell>
                          </TableRow>
                        )}
                      </TableBody>
                    </Table>
                  </TableContainer>
                </AccordionDetails>
              </Accordion>
            </Box>
          )}
        </DrawerContent>
      </Drawer>
    </Box>
  );
};

export default Discover;
