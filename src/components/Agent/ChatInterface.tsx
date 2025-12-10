import React, { useRef, useEffect, useState } from "react";
import type {
  Message,
  InvestigationStatus,
  AgentPlan,
} from "../../types/agent";
import { MessageBubble } from "./MessageBubble";
import { AgentStep } from "./AgentStep";
import {
  Box,
  TextField,
  IconButton,
  Typography,
  Paper,
  Container,
} from "@mui/material";
import SendIcon from "@mui/icons-material/Send";
import StopCircleIcon from "@mui/icons-material/StopCircle";
import AutoAwesomeIcon from "@mui/icons-material/AutoAwesome";

interface ChatInterfaceProps {
  messages: Message[];
  status: InvestigationStatus;
  currentThought: string;
  currentHypothesis: string;
  currentQuery?: AgentPlan["query"];
  onStartInvestigation: (problem: string) => void;
  onStop: () => void;
}

export const ChatInterface: React.FC<ChatInterfaceProps> = ({
  messages,
  status,
  currentThought,
  currentHypothesis,
  currentQuery,
  onStartInvestigation,
  onStop,
}) => {
  const [input, setInput] = useState("");
  const bottomRef = useRef<HTMLDivElement>(null);
  const isBusy =
    status !== "idle" && status !== "complete" && status !== "error";

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, currentThought, status]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim()) return;
    onStartInvestigation(input);
    setInput("");
  };

  return (
    <Box
      sx={{
        display: "flex",
        flexDirection: "column",
        height: "100%",
        overflow: "hidden",
      }}
    >
      {/* Messages Area */}
      <Box sx={{ flex: 1, overflowY: "auto", p: 3, pb: 15 }}>
        <Container maxWidth="md">
          {messages.length === 0 && (
            <Box
              sx={{
                height: "100%",
                display: "flex",
                flexDirection: "column",
                alignItems: "center",
                justifyContent: "center",
                textAlign: "center",
                opacity: 0.7,
                py: 10,
              }}
            >
              <Box
                sx={{
                  width: 80,
                  height: 80,
                  borderRadius: 4,
                  bgcolor: "action.hover",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  mb: 3,
                }}
              >
                <AutoAwesomeIcon sx={{ fontSize: 40, color: "primary.main" }} />
              </Box>
              <Typography
                variant="h4"
                gutterBottom
                sx={{
                  background:
                    "linear-gradient(45deg, #2196F3 30%, #21CBF3 90%)",
                  WebkitBackgroundClip: "text",
                  WebkitTextFillColor: "transparent",
                }}
              >
                Hello, Human
              </Typography>
              <Typography
                variant="h6"
                color="text.secondary"
                sx={{ maxWidth: 400 }}
              >
                I can help you troubleshoot Kubernetes cluster events. What
                seems to be the problem?
              </Typography>
            </Box>
          )}

          {messages
            .filter(
              (msg) =>
                msg.role !== "system" ||
                msg.content.startsWith("Query executed. Result:") ||
                msg.content.startsWith("Error:") ||
                msg.content.startsWith("AI did not provide") ||
                msg.content.startsWith("Maximum turns reached"),
            )
            .map((msg, idx) => (
              <MessageBubble key={idx} message={msg} />
            ))}

          <AgentStep
            status={status}
            thought={currentThought}
            hypothesis={currentHypothesis}
            query={currentQuery}
          />
          <div ref={bottomRef} />
        </Container>
      </Box>

      {/* Input Area */}
      <Box
        sx={{
          position: "absolute",
          bottom: 0,
          left: 0,
          right: 0,
          p: 3,
          background: (theme) =>
            `linear-gradient(to top, ${theme.palette.background.default} 80%, transparent)`,
          backdropFilter: "blur(8px)",
          zIndex: 10,
        }}
      >
        <Container maxWidth="md">
          <Paper
            component="form"
            onSubmit={handleSubmit}
            elevation={0}
            variant="outlined"
            sx={{
              p: "2px 4px",
              display: "flex",
              alignItems: "center",
              borderRadius: 4,
              bgcolor: "action.hover",
              border: 1,
              borderColor: "transparent",
              "&:focus-within": {
                borderColor: "primary.main",
                bgcolor: "background.paper",
              },
              transition: "all 0.2s",
            }}
          >
            <TextField
              fullWidth
              variant="standard"
              placeholder={
                isBusy
                  ? "Investigation in progress..."
                  : "Describe the problem (e.g., 'Pods are failing on node-1')..."
              }
              value={input}
              onChange={(e) => setInput(e.target.value)}
              disabled={isBusy}
              InputProps={{
                disableUnderline: true,
                sx: { px: 3, py: 1.5, fontSize: "1rem" },
              }}
            />
            <Box sx={{ p: 1 }}>
              {isBusy ? (
                <IconButton
                  onClick={onStop}
                  color="error"
                  title="Stop Investigation"
                >
                  <StopCircleIcon />
                </IconButton>
              ) : (
                <IconButton
                  type="submit"
                  color="primary"
                  disabled={!input.trim()}
                >
                  <SendIcon />
                </IconButton>
              )}
            </Box>
          </Paper>
          <Typography
            variant="caption"
            display="block"
            align="center"
            color="text.secondary"
            sx={{ mt: 1 }}
          >
            Kabinet can make mistakes. Consider checking important information.
          </Typography>
        </Container>
      </Box>
    </Box>
  );
};
