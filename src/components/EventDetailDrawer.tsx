import React from "react";
import {
  Typography,
  Box,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableRow,
  Chip,
  Drawer,
  IconButton,
  alpha,
} from "@mui/material";
import { styled } from "@mui/material/styles";
import { Close } from "@mui/icons-material";
import { Link } from "./Link";
import type { EventResult } from "../types/events";

interface EventDetailDrawerProps {
  open: boolean;
  event: EventResult | null;
  onClose: () => void;
}

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

const ClickableCell: React.FC<{
  field: string;
  value: string;
  onClose: () => void;
}> = ({ field, value, onClose }) => {
  if (!value) {
    return <TableCell />;
  }
  const query = `${field}='${value}'`;
  return (
    <TableCell>
      <Link page="discover" params={{ where: query }} onClick={onClose}>
        {value}
      </Link>
    </TableCell>
  );
};

const ClickableChipCell: React.FC<{
  field: string;
  value: string;
  onClose: () => void;
}> = ({ field, value, onClose }) => {
  if (!value) {
    return <TableCell />;
  }
  const query = `${field}='${value}'`;
  return (
    <TableCell>
      <Link page="discover" params={{ where: query }} onClick={onClose}>
        <EventTypeChip label={value} size="small" eventType={value} />
      </Link>
    </TableCell>
  );
};

const EventDetailDrawer: React.FC<EventDetailDrawerProps> = ({
  open,
  event,
  onClose,
}) => {
  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  return (
    <Drawer anchor="right" open={open} onClose={onClose} variant="temporary">
      <DrawerContent>
        <DrawerHeader>
          <Typography variant="h6" sx={{ fontWeight: 600 }}>
            Event Details
          </Typography>
          <IconButton onClick={onClose} size="small">
            <Close />
          </IconButton>
        </DrawerHeader>

        {event && (
          <Box sx={{ p: 3 }}>
            {/* Event Summary */}
            <Box sx={{ mb: 4 }}>
              <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
                Event Summary
              </Typography>
              <TableContainer>
                <Table size="small">
                  <TableBody>
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600, width: "40%" }}
                      >
                        Type
                      </TableCell>
                      <ClickableChipCell
                        field="type"
                        value={event.type}
                        onClose={onClose}
                      />
                    </TableRow>
                    <TableRow>
                      <TableCell component="th" scope="row">
                        Reason
                      </TableCell>
                      <ClickableCell
                        field="reason"
                        value={event.reason}
                        onClose={onClose}
                      />
                    </TableRow>
                    <TableRow>
                      <TableCell component="th" scope="row">
                        Object Kind
                      </TableCell>
                      <ClickableCell
                        field="involvedObject.kind"
                        value={event.involvedObject.kind}
                        onClose={onClose}
                      />
                    </TableRow>
                    <TableRow>
                      <TableCell component="th" scope="row">
                        Object Namespace
                      </TableCell>
                      <ClickableCell
                        field="involvedObject.namespace"
                        value={event.involvedObject.namespace || "default"}
                        onClose={onClose}
                      />
                    </TableRow>
                    <TableRow>
                      <TableCell component="th" scope="row">
                        Object Name
                      </TableCell>
                      <ClickableCell
                        field="involvedObject.name"
                        value={event.involvedObject.name}
                        onClose={onClose}
                      />
                    </TableRow>
                  </TableBody>
                </Table>
              </TableContainer>
            </Box>

            {/* Message */}
            <Box sx={{ mb: 4 }}>
              <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
                Message
              </Typography>
              <Typography
                variant="body1"
                sx={{
                  lineHeight: 1.6,
                  fontFamily: "monospace",
                  backgroundColor: alpha("#f5f5f5", 0.8),
                  p: 2,
                  borderRadius: 1,
                  border: `1px solid ${alpha("#e0e0e0", 0.3)}`,
                }}
              >
                {event.message}
              </Typography>
            </Box>

            {/* Timestamps */}
            <Box sx={{ mb: 4 }}>
              <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
                Timestamps
              </Typography>
              <TableContainer>
                <Table size="small">
                  <TableBody>
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600, width: "40%" }}
                      >
                        Created
                      </TableCell>
                      <TableCell>
                        {formatTimestamp(event.metadata.creationTimestamp)}
                      </TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600 }}
                      >
                        First Timestamp
                      </TableCell>
                      <TableCell>
                        {formatTimestamp(event.firstTimestamp)}
                      </TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600 }}
                      >
                        Last Timestamp
                      </TableCell>
                      <TableCell>
                        {formatTimestamp(event.lastTimestamp)}
                      </TableCell>
                    </TableRow>
                    {event.eventTime && (
                      <TableRow>
                        <TableCell
                          component="th"
                          scope="row"
                          sx={{ fontWeight: 600 }}
                        >
                          Event Time
                        </TableCell>
                        <TableCell>
                          {formatTimestamp(event.eventTime)}
                        </TableCell>
                      </TableRow>
                    )}
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600 }}
                      >
                        Count
                      </TableCell>
                      <TableCell>{event.count}</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </TableContainer>
            </Box>

            {/* Event Metadata */}
            <Box sx={{ mb: 4 }}>
              <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
                Event Metadata
              </Typography>
              <TableContainer>
                <Table size="small">
                  <TableBody>
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600, width: "40%" }}
                      >
                        Event Name
                      </TableCell>
                      <TableCell
                        sx={{
                          fontFamily: "monospace",
                          fontSize: "0.875rem",
                        }}
                      >
                        {event.metadata.name}
                      </TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600, width: "40%" }}
                      >
                        Event UID
                      </TableCell>
                      <TableCell
                        sx={{
                          fontFamily: "monospace",
                          fontSize: "0.875rem",
                        }}
                      >
                        {event.metadata.uid}
                      </TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600 }}
                      >
                        Event Kind
                      </TableCell>
                      <TableCell>{event.kind}</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600 }}
                      >
                        API Version
                      </TableCell>
                      <TableCell>{event.apiVersion}</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600 }}
                      >
                        Resource Version
                      </TableCell>
                      <TableCell
                        sx={{
                          fontFamily: "monospace",
                          fontSize: "0.875rem",
                        }}
                      >
                        {event.metadata.resourceVersion}
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
                      <ClickableCell
                        field="metadata.namespace"
                        value={event.metadata.namespace || "default"}
                        onClose={onClose}
                      />
                    </TableRow>
                    {event.action && (
                      <TableRow>
                        <TableCell
                          component="th"
                          scope="row"
                          sx={{ fontWeight: 600 }}
                        >
                          Action
                        </TableCell>
                        <TableCell>{event.action}</TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </TableContainer>
            </Box>

            {/* Involved Object */}
            <Box sx={{ mb: 4 }}>
              <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
                Involved Object
              </Typography>
              <TableContainer>
                <Table size="small">
                  <TableBody>
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600, width: "40%" }}
                      >
                        Kind
                      </TableCell>
                      <ClickableCell
                        field="involvedObject.kind"
                        value={event.involvedObject.kind}
                        onClose={onClose}
                      />
                    </TableRow>
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600 }}
                      >
                        Name
                      </TableCell>
                      <ClickableCell
                        field="involvedObject.name"
                        value={event.involvedObject.name}
                        onClose={onClose}
                      />
                    </TableRow>
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600 }}
                      >
                        Namespace
                      </TableCell>
                      <ClickableCell
                        field="involvedObject.namespace"
                        value={event.involvedObject.namespace || "default"}
                        onClose={onClose}
                      />
                    </TableRow>
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600 }}
                      >
                        UID
                      </TableCell>
                      <TableCell
                        sx={{
                          fontFamily: "monospace",
                          fontSize: "0.875rem",
                        }}
                      >
                        {event.involvedObject.uid}
                      </TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600 }}
                      >
                        API Version
                      </TableCell>
                      <TableCell>{event.involvedObject.apiVersion}</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600 }}
                      >
                        Resource Version
                      </TableCell>
                      <TableCell
                        sx={{
                          fontFamily: "monospace",
                          fontSize: "0.875rem",
                        }}
                      >
                        {event.involvedObject.resourceVersion}
                      </TableCell>
                    </TableRow>
                    {event.involvedObject.fieldPath && (
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
                          {event.involvedObject.fieldPath}
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </TableContainer>
            </Box>

            {/* Source Information */}
            <Box sx={{ mb: 4 }}>
              <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
                Source Information
              </Typography>
              <TableContainer>
                <Table size="small">
                  <TableBody>
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600, width: "40%" }}
                      >
                        Source Component
                      </TableCell>
                      <ClickableCell
                        field="source.component"
                        value={event.source.component}
                        onClose={onClose}
                      />
                    </TableRow>
                    <TableRow>
                      <TableCell
                        component="th"
                        scope="row"
                        sx={{ fontWeight: 600 }}
                      >
                        Source Host
                      </TableCell>
                      <ClickableCell
                        field="source.host"
                        value={event.source.host}
                        onClose={onClose}
                      />
                    </TableRow>
                    {event.reportingComponent && (
                      <TableRow>
                        <TableCell
                          component="th"
                          scope="row"
                          sx={{ fontWeight: 600 }}
                        >
                          Reporting Component
                        </TableCell>
                        <TableCell>{event.reportingComponent}</TableCell>
                      </TableRow>
                    )}
                    {event.reportingInstance && (
                      <TableRow>
                        <TableCell
                          component="th"
                          scope="row"
                          sx={{ fontWeight: 600 }}
                        >
                          Reporting Instance
                        </TableCell>
                        <TableCell>{event.reportingInstance}</TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </TableContainer>
            </Box>

            {/* Series Information */}
            {event.series && (
              <Box sx={{ mb: 4 }}>
                <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
                  Series Information
                </Typography>
                <TableContainer>
                  <Table size="small">
                    <TableBody>
                      <TableRow>
                        <TableCell
                          component="th"
                          scope="row"
                          sx={{ fontWeight: 600, width: "40%" }}
                        >
                          Series Count
                        </TableCell>
                        <TableCell>{event.series.count}</TableCell>
                      </TableRow>
                      <TableRow>
                        <TableCell
                          component="th"
                          scope="row"
                          sx={{ fontWeight: 600 }}
                        >
                          Last Observed Time
                        </TableCell>
                        <TableCell>
                          {formatTimestamp(event.series.lastObservedTime)}
                        </TableCell>
                      </TableRow>
                    </TableBody>
                  </Table>
                </TableContainer>
              </Box>
            )}

            {/* Related Object */}
            {event.related && (
              <Box sx={{ mb: 4 }}>
                <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
                  Related Object
                </Typography>
                <TableContainer>
                  <Table size="small">
                    <TableBody>
                      <TableRow>
                        <TableCell
                          component="th"
                          scope="row"
                          sx={{ fontWeight: 600, width: "40%" }}
                        >
                          Kind
                        </TableCell>
                        <TableCell>{event.related.kind}</TableCell>
                      </TableRow>
                      <TableRow>
                        <TableCell
                          component="th"
                          scope="row"
                          sx={{ fontWeight: 600 }}
                        >
                          Name
                        </TableCell>
                        <TableCell>{event.related.name}</TableCell>
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
                          {event.related.namespace || "default"}
                        </TableCell>
                      </TableRow>
                      <TableRow>
                        <TableCell
                          component="th"
                          scope="row"
                          sx={{ fontWeight: 600 }}
                        >
                          UID
                        </TableCell>
                        <TableCell
                          sx={{
                            fontFamily: "monospace",
                            fontSize: "0.875rem",
                          }}
                        >
                          {event.related.uid}
                        </TableCell>
                      </TableRow>
                      <TableRow>
                        <TableCell
                          component="th"
                          scope="row"
                          sx={{ fontWeight: 600 }}
                        >
                          API Version
                        </TableCell>
                        <TableCell>{event.related.apiVersion}</TableCell>
                      </TableRow>
                      <TableRow>
                        <TableCell
                          component="th"
                          scope="row"
                          sx={{ fontWeight: 600 }}
                        >
                          Resource Version
                        </TableCell>
                        <TableCell
                          sx={{
                            fontFamily: "monospace",
                            fontSize: "0.875rem",
                          }}
                        >
                          {event.related.resourceVersion}
                        </TableCell>
                      </TableRow>
                      {event.related.fieldPath && (
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
                            {event.related.fieldPath}
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </TableContainer>
              </Box>
            )}
          </Box>
        )}
      </DrawerContent>
    </Drawer>
  );
};

export default EventDetailDrawer;
