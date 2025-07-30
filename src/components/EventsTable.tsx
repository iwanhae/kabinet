import React from "react";
import {
  Typography,
  Box,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableRow,
  TableHead,
  Chip,
  alpha,
} from "@mui/material";
import { styled } from "@mui/material/styles";
import type { EventResult } from "../types/events";

interface EventsTableProps {
  events: EventResult[];
  onEventClick: (event: EventResult) => void;
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

const EventsTable: React.FC<EventsTableProps> = ({ events, onEventClick }) => {
  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  if (events.length === 0) {
    return null;
  }

  return (
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
          Found {events.length} events
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
            {events.map((event, index) => (
              <StyledTableRow
                key={event.metadata.uid || index}
                eventType={event.type}
                onClick={() => onEventClick(event)}
              >
                <TableCell sx={{ py: 1.5, width: "90px", minWidth: "90px" }}>
                  <EventTypeChip
                    label={event.type}
                    size="small"
                    eventType={event.type}
                  />
                </TableCell>
                <TableCell sx={{ py: 1.5, width: "150px", minWidth: "150px" }}>
                  <Typography
                    variant="body2"
                    color="text.secondary"
                    sx={{ fontSize: "0.8rem" }}
                  >
                    {formatTimestamp(event.metadata.creationTimestamp)}
                  </Typography>
                </TableCell>
                <TableCell sx={{ py: 1.5, width: "120px", minWidth: "120px" }}>
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
                <TableCell sx={{ py: 1.5, width: "200px", minWidth: "200px" }}>
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
  );
};

export default EventsTable;
