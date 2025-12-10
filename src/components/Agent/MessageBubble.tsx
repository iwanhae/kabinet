import React from "react";
import type { Message, AgentPlan } from "../../types/agent";
import { DataVisualizer } from "./DataVisualizer";
import {
  Box,
  Paper,
  Typography,
  Avatar,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  alpha,
} from "@mui/material";
import PersonIcon from "@mui/icons-material/Person";
import AutoAwesomeIcon from "@mui/icons-material/AutoAwesome";
import TerminalIcon from "@mui/icons-material/Terminal";
import FormatQuoteIcon from "@mui/icons-material/FormatQuote";
import TipsAndUpdatesIcon from "@mui/icons-material/TipsAndUpdates";
import StorageIcon from "@mui/icons-material/Storage";
import CheckCircleOutlineIcon from "@mui/icons-material/CheckCircleOutline";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";

interface MessageBubbleProps {
  message: Message;
}

export const MessageBubble: React.FC<MessageBubbleProps> = ({ message }) => {
  const isUser = message.role === "user";
  const isSystem = message.role === "system";
  const isQueryResult = message.content.startsWith("Query executed. Result:");

  if (isSystem) {
    if (isQueryResult) {
      const jsonContent = message.content.replace(
        "Query executed. Result:\n",
        "",
      );
      return (
        <Accordion
          variant="outlined"
          sx={{ mb: 2, bgcolor: "background.paper" }}
        >
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
              <StorageIcon color="primary" fontSize="small" />
              <Typography variant="subtitle2" sx={{ fontWeight: "bold" }}>
                Query Result
              </Typography>
            </Box>
          </AccordionSummary>
          <AccordionDetails>
            <Box
              component="pre"
              sx={{
                m: 0,
                p: 2,
                bgcolor: "grey.900",
                color: "success.light",
                borderRadius: 1,
                overflowX: "auto",
                fontSize: "0.75rem",
                fontFamily: "monospace",
              }}
            >
              {jsonContent}
            </Box>
          </AccordionDetails>
        </Accordion>
      );
    }

    return (
      <Paper
        variant="outlined"
        sx={{
          p: 2,
          mb: 2,
          display: "flex",
          gap: 2,
          bgcolor: "action.hover",
          fontFamily: "monospace",
        }}
      >
        <TerminalIcon color="action" />
        <Typography
          variant="body2"
          sx={{ whiteSpace: "pre-wrap", opacity: 0.9 }}
        >
          {message.content}
        </Typography>
      </Paper>
    );
  }

  const content = message.content;
  let parsedPlan: AgentPlan | null = null;

  if (!isUser) {
    try {
      parsedPlan = JSON.parse(message.content);
    } catch (e) {
      // Not JSON, render as text
    }
  }

  return (
    <Box
      sx={{
        display: "flex",
        gap: 2,
        flexDirection: isUser ? "row-reverse" : "row",
        mb: 3,
      }}
    >
      <Avatar
        sx={{
          bgcolor: isUser ? "primary.main" : "transparent",
          color: isUser ? "primary.contrastText" : "text.primary",
          width: 32,
          height: 32,
          border: isUser ? "none" : "1px solid",
          borderColor: "divider",
        }}
      >
        {isUser ? (
          <PersonIcon fontSize="small" />
        ) : (
          <AutoAwesomeIcon fontSize="small" />
        )}
      </Avatar>

      <Paper
        elevation={0}
        variant="outlined"
        sx={{
          p: 2,
          maxWidth: "85%",
          borderRadius: 3,
          bgcolor: isUser ? "primary.light" : "background.paper",
          color: isUser ? "primary.contrastText" : "text.primary",
          border: isUser ? "none" : "1px solid",
          borderColor: "divider",
        }}
      >
        {parsedPlan ? (
          <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
            {parsedPlan.thought && (
              <Box sx={{ display: "flex", gap: 1.5, color: "text.secondary" }}>
                <FormatQuoteIcon
                  fontSize="small"
                  sx={{ mt: -0.2, opacity: 0.5 }}
                />
                <Typography
                  variant="body2"
                  sx={{ fontStyle: "italic", lineHeight: 1.6 }}
                >
                  {parsedPlan.thought}
                </Typography>
              </Box>
            )}

            {parsedPlan.hypothesis && (
              <Box
                sx={{
                  pl: 2,
                  borderLeft: "3px solid",
                  borderColor: "warning.main",
                  bgcolor: (theme) => alpha(theme.palette.warning.main, 0.05),
                  p: 1.5,
                  borderRadius: 1,
                }}
              >
                <Box
                  sx={{
                    display: "flex",
                    alignItems: "center",
                    gap: 1,
                    mb: 0.5,
                  }}
                >
                  <TipsAndUpdatesIcon fontSize="small" color="warning" />
                  <Typography
                    variant="caption"
                    sx={{
                      fontWeight: 700,
                      letterSpacing: 0.5,
                      color: "warning.main",
                    }}
                  >
                    HYPOTHESIS
                  </Typography>
                </Box>
                <Typography variant="body2">{parsedPlan.hypothesis}</Typography>
              </Box>
            )}

            {parsedPlan.query && (
              <Box>
                <Box
                  sx={{ display: "flex", alignItems: "center", gap: 1, mb: 1 }}
                >
                  <TerminalIcon fontSize="small" color="action" />
                  <Typography
                    variant="caption"
                    sx={{ fontWeight: 600, color: "text.secondary" }}
                  >
                    SQL QUERY
                  </Typography>
                </Box>
                <Box
                  component="div"
                  sx={{
                    bgcolor: "grey.900",
                    color: "grey.100",
                    p: 2,
                    borderRadius: 2,
                    fontFamily: "monospace",
                    fontSize: "0.8rem",
                    overflowX: "auto",
                    border: "1px solid",
                    borderColor: "grey.800",
                  }}
                >
                  {parsedPlan.query.sql}
                </Box>
              </Box>
            )}

            {parsedPlan.final_analysis && (
              <Box
                sx={{
                  borderTop: "1px dashed",
                  borderColor: "divider",
                  pt: 2,
                  mt: 1,
                }}
              >
                <Box
                  sx={{ display: "flex", gap: 1, mb: 1, alignItems: "center" }}
                >
                  <CheckCircleOutlineIcon color="success" fontSize="small" />
                  <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                    Conclusion
                  </Typography>
                </Box>
                <Typography variant="body1" sx={{ lineHeight: 1.6 }}>
                  {parsedPlan.final_analysis}
                </Typography>
              </Box>
            )}

            {parsedPlan.data && <DataVisualizer data={parsedPlan.data} />}
          </Box>
        ) : (
          <Typography variant="body1" sx={{ whiteSpace: "pre-wrap" }}>
            {content}
          </Typography>
        )}
      </Paper>
    </Box>
  );
};
