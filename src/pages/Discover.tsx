import React, { useState, useEffect } from "react";
import { Typography, Box, Alert, CircularProgress } from "@mui/material";
import { useSearch } from "wouter";
import { invalidateEventsQuery, useEventsQuery } from "../hooks/useEventsQuery";
import { useQueryParams } from "../hooks/useUrlParams";
import type { EventResult } from "../types/events";
import QueryForm from "../components/QueryForm";
import EventsTable from "../components/EventsTable";
import EventDetailDrawer from "../components/EventDetailDrawer";
import EventsTimelineChart from "../components/EventsTimelineChart";

const Discover: React.FC = () => {
  const { setWhereClause: updateUrlWhereClause } = useQueryParams();
  const search = useSearch();
  const [whereClause, setWhereClause] = useState("");
  const [executedQuery, setExecutedQuery] = useState<string | null>(null);
  const [selectedEvent, setSelectedEvent] = useState<EventResult | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);

  // URL의 where 파라미터를 읽어서 executedQuery 초기값으로 설정
  useEffect(() => {
    const searchParams = new URLSearchParams(search);
    const whereParam = searchParams.get("where");

    if (whereParam) {
      setExecutedQuery(whereParam);
      setWhereClause(whereParam);
    }
  }, [search]);

  // 쿼리가 실행된 경우에만 데이터를 가져옴
  const query = executedQuery
    ? `SELECT * FROM $events WHERE ${executedQuery} ORDER BY metadata.creationTimestamp DESC LIMIT 100`
    : null;

  const { data, error, isLoading } = useEventsQuery<EventResult>(query);

  const handleExecuteQuery = () => {
    const trimmedWhereClause = whereClause.trim() || "1=1";
    setExecutedQuery(trimmedWhereClause);

    // URL의 where 파라미터 업데이트 (기존 파라미터 유지)
    updateUrlWhereClause(trimmedWhereClause);

    invalidateEventsQuery();
  };

  const handleEventClick = (event: EventResult) => {
    setSelectedEvent(event);
    setDrawerOpen(true);
  };

  const handleDrawerClose = () => {
    setDrawerOpen(false);
    setSelectedEvent(null);
  };

  return (
    <Box>
      <QueryForm
        whereClause={whereClause}
        onWhereClauseChange={setWhereClause}
        onExecuteQuery={handleExecuteQuery}
        isLoading={isLoading}
      />

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
        <>
          <EventsTimelineChart where={executedQuery || "1=1"} />
          <EventsTable events={data} onEventClick={handleEventClick} />
        </>
      )}

      <EventDetailDrawer
        open={drawerOpen}
        event={selectedEvent}
        onClose={handleDrawerClose}
      />
    </Box>
  );
};

export default Discover;
