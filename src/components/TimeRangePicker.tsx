import { useState } from "react";
import {
  Button,
  Popover,
  List,
  ListItemButton,
  ListItemText,
  TextField,
  Box,
  Typography,
  styled,
  Collapse,
  Link,
  IconButton,
  Tooltip,
} from "@mui/material";
import {
  AccessTime,
  ArrowDropDown,
  HelpOutline,
  Refresh,
} from "@mui/icons-material";
import { DateTimePicker } from "@mui/x-date-pickers/DateTimePicker";
import { LocalizationProvider } from "@mui/x-date-pickers/LocalizationProvider";
import { AdapterDayjs } from "@mui/x-date-pickers/AdapterDayjs";
import dayjs, { Dayjs } from "dayjs";
import timezone from "dayjs/plugin/timezone";
import utc from "dayjs/plugin/utc";
import {
  useTimeRangeFromUrl,
  useTimeRangeStore,
} from "../stores/timeRangeStore";

// Configure dayjs plugins
dayjs.extend(utc);
dayjs.extend(timezone);

const quickRanges = [
  { label: "Last 5 minutes", from: "now-5m", to: "now" },
  { label: "Last 15 minutes", from: "now-15m", to: "now" },
  { label: "Last 30 minutes", from: "now-30m", to: "now" },
  { label: "Last 1 hour", from: "now-1h", to: "now" },
  { label: "Last 3 hours", from: "now-3h", to: "now" },
  { label: "Last 6 hours", from: "now-6h", to: "now" },
  { label: "Last 12 hours", from: "now-12h", to: "now" },
  { label: "Last 24 hours", from: "now-24h", to: "now" },
  { label: "Last 3 days", from: "now-3d", to: "now" },
  { label: "Last 7 days", from: "now-7d", to: "now" },
  { label: "Last 30 days", from: "now-30d", to: "now" },
];

const TimeRangePickerContainer = styled(Box)({
  display: "flex",
  alignItems: "center",
  justifyContent: "flex-end",
  gap: "8px",
});

const TimeRangeButton = styled(Button)(({ theme }) => ({
  color: theme.palette.text.primary,
  borderColor: theme.palette.divider,
}));

const PopoverContent = styled(Box)(({ theme }) => ({
  display: "flex",
  width: "650px",
  height: "400px",
  backgroundColor: theme.palette.background.paper,
}));

const QuickRangesContainer = styled(Box)(({ theme }) => ({
  width: "220px",
  borderRight: `1px solid ${theme.palette.divider}`,
  display: "flex",
  flexDirection: "column",
}));

const QuickRangesHeader = styled(Box)(({ theme }) => ({
  padding: theme.spacing(2),
  borderBottom: `1px solid ${theme.palette.divider}`,
  backgroundColor: theme.palette.grey[50],
  ...(theme.palette.mode === "dark" && {
    backgroundColor: theme.palette.grey[900],
  }),
}));

const QuickRangesList = styled(Box)({
  flex: 1,
  overflowY: "auto",
  maxHeight: "300px",
});

const CustomTimeRangeContainer = styled(Box)(({ theme }) => ({
  flex: 1,
  padding: theme.spacing(3),
  display: "flex",
  flexDirection: "column",
  gap: theme.spacing(2),
}));

const TimezoneInfo = styled(Box)(({ theme }) => ({
  padding: theme.spacing(1, 2),
  backgroundColor: theme.palette.grey[100],
  borderRadius: theme.spacing(1),
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  fontSize: "0.875rem",
  color: theme.palette.text.secondary,
  ...(theme.palette.mode === "dark" && {
    backgroundColor: theme.palette.grey[800],
  }),
}));

const HelpText = styled(Box)(({ theme }) => ({
  padding: theme.spacing(1.5),
  backgroundColor: theme.palette.grey[50],
  borderRadius: theme.spacing(1),
  fontSize: "0.75rem",
  color: theme.palette.text.secondary,
  ...(theme.palette.mode === "dark" && {
    backgroundColor: theme.palette.grey[900],
  }),
}));

// Helper functions for date/time conversion
const parseTimeString = (timeStr: string): Dayjs | null => {
  if (timeStr === "now") {
    return dayjs();
  }
  if (timeStr.startsWith("now-")) {
    const duration = timeStr.substring(4);
    const match = duration.match(/^(\d+)([mhd])$/);
    if (match) {
      const [, amount, unit] = match;
      const unitMap: { [key: string]: "minute" | "hour" | "day" } = {
        m: "minute",
        h: "hour",
        d: "day",
      };
      return dayjs().subtract(parseInt(amount), unitMap[unit]);
    }
  }
  // Try parsing as ISO string
  const parsed = dayjs(timeStr);
  return parsed.isValid() ? parsed : null;
};

const formatTimeString = (date: Dayjs): string => {
  return date.toISOString();
};

const isRelativeTime = (timeStr: string): boolean => {
  return timeStr === "now" || timeStr.startsWith("now-");
};

// Get current timezone info
const getCurrentTimezone = () => {
  const timezone = dayjs.tz.guess();
  const offset = dayjs().format("Z");
  const abbreviation = dayjs().format("z");
  return {
    timezone,
    offset,
    abbreviation,
    displayName: `${timezone} (UTC${offset})`,
  };
};

export const TimeRangePicker = () => {
  const [anchorEl, setAnchorEl] = useState<HTMLButtonElement | null>(null);
  const { from, to, rawFrom, rawTo, refreshTimeRange } = useTimeRangeStore();
  const setTimeRangeWithUrl = useTimeRangeFromUrl();

  const [tempFromDate, setTempFromDate] = useState<Dayjs | null>(
    parseTimeString(from),
  );
  const [tempToDate, setTempToDate] = useState<Dayjs | null>(
    parseTimeString(to),
  );
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [tempFromText, setTempFromText] = useState(from);
  const [tempToText, setTempToText] = useState(to);

  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    setAnchorEl(event.currentTarget);
    setTempFromDate(parseTimeString(from));
    setTempToDate(parseTimeString(to));
    setTempFromText(from);
    setTempToText(to);
    setShowAdvanced(isRelativeTime(from) || isRelativeTime(to));
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const handleQuickRangeSelect = (from: string, to: string) => {
    setTimeRangeWithUrl(from, to);
    handleClose();
  };

  const handleDateChange = (field: "from" | "to", date: Dayjs | null) => {
    if (date && date.isValid()) {
      if (field === "from") {
        setTempFromDate(date);
        setTempFromText(formatTimeString(date));
      } else {
        setTempToDate(date);
        setTempToText(formatTimeString(date));
      }
    }
  };

  const handleTextChange = (field: "from" | "to", value: string) => {
    if (field === "from") {
      setTempFromText(value);
      const parsed = parseTimeString(value);
      if (parsed) {
        setTempFromDate(parsed);
      }
    } else {
      setTempToText(value);
      const parsed = parseTimeString(value);
      if (parsed) {
        setTempToDate(parsed);
      }
    }
  };

  const handleApply = () => {
    if (showAdvanced) {
      // Use text values for advanced mode (supports relative time)
      setTimeRangeWithUrl(tempFromText, tempToText);
    } else {
      // Use date picker values for simple mode
      if (tempFromDate && tempToDate) {
        setTimeRangeWithUrl(
          formatTimeString(tempFromDate),
          formatTimeString(tempToDate),
        );
      }
    }
    handleClose();
  };

  const handleRefresh = () => {
    refreshTimeRange();
  };

  const open = Boolean(anchorEl);
  const id = open ? "time-range-popover" : undefined;
  const timezoneInfo = getCurrentTimezone();

  const displayLabel =
    quickRanges.find((r) => r.from === rawFrom && r.to === "now")?.label ||
    `${from} to ${to}`;

  const isRelative = isRelativeTime(rawFrom) || isRelativeTime(rawTo);

  return (
    <LocalizationProvider dateAdapter={AdapterDayjs}>
      <TimeRangePickerContainer>
        <TimeRangeButton
          aria-describedby={id}
          variant="outlined"
          onClick={handleClick}
          startIcon={<AccessTime />}
          endIcon={<ArrowDropDown />}
          size="small"
        >
          <Box
            sx={{
              display: "flex",
              flexDirection: "column",
              alignItems: "flex-start",
            }}
          >
            <Typography variant="body2" component="span">
              {displayLabel}
            </Typography>
            <Typography
              variant="caption"
              component="span"
              sx={{ opacity: 0.7 }}
            >
              {timezoneInfo.abbreviation || timezoneInfo.offset}
            </Typography>
          </Box>
        </TimeRangeButton>
        {isRelative && (
          <Tooltip title="Refresh time range">
            <IconButton onClick={handleRefresh} size="small">
              <Refresh />
            </IconButton>
          </Tooltip>
        )}
        <Popover
          id={id}
          open={open}
          anchorEl={anchorEl}
          onClose={handleClose}
          anchorOrigin={{
            vertical: "bottom",
            horizontal: "right",
          }}
          transformOrigin={{
            vertical: "top",
            horizontal: "right",
          }}
          PaperProps={{
            elevation: 8,
            sx: { borderRadius: 2 },
          }}
        >
          <PopoverContent>
            <QuickRangesContainer>
              <QuickRangesHeader>
                <Typography
                  variant="subtitle2"
                  fontWeight={600}
                  color="text.secondary"
                >
                  Quick ranges
                </Typography>
              </QuickRangesHeader>
              <QuickRangesList>
                <List dense>
                  {quickRanges.map((range) => (
                    <ListItemButton
                      key={range.label}
                      onClick={() =>
                        handleQuickRangeSelect(range.from, range.to)
                      }
                      selected={rawFrom === range.from && rawTo === range.to}
                      sx={{
                        py: 1,
                        px: 2,
                        mx: 1,
                        my: 0.5,
                        borderRadius: 1,
                        "&.Mui-selected": {
                          backgroundColor: "primary.main",
                          color: "white",
                          "&:hover": {
                            backgroundColor: "primary.dark",
                          },
                        },
                      }}
                    >
                      <ListItemText
                        primary={range.label}
                        primaryTypographyProps={{
                          fontSize: "0.875rem",
                          fontWeight:
                            rawFrom === range.from && rawTo === range.to
                              ? 600
                              : 400,
                        }}
                      />
                    </ListItemButton>
                  ))}
                </List>
              </QuickRangesList>
            </QuickRangesContainer>

            <CustomTimeRangeContainer>
              <Typography variant="h6" gutterBottom fontWeight={600}>
                Custom time range
              </Typography>

              {!showAdvanced ? (
                <>
                  <DateTimePicker
                    label="From"
                    value={tempFromDate}
                    onChange={(date) => handleDateChange("from", date)}
                    slotProps={{
                      textField: {
                        fullWidth: true,
                        size: "small",
                      },
                    }}
                  />
                  <DateTimePicker
                    label="To"
                    value={tempToDate}
                    onChange={(date) => handleDateChange("to", date)}
                    slotProps={{
                      textField: {
                        fullWidth: true,
                        size: "small",
                      },
                    }}
                  />
                </>
              ) : (
                <>
                  <TextField
                    label="From"
                    value={tempFromText}
                    onChange={(e) => handleTextChange("from", e.target.value)}
                    fullWidth
                    size="small"
                    placeholder="e.g., now-1h, 2025-01-01T00:00:00Z"
                  />
                  <TextField
                    label="To"
                    value={tempToText}
                    onChange={(e) => handleTextChange("to", e.target.value)}
                    fullWidth
                    size="small"
                    placeholder="e.g., now, 2025-01-01T12:00:00Z"
                  />
                </>
              )}

              <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                <HelpOutline sx={{ fontSize: 16, color: "text.secondary" }} />
                <Link
                  component="button"
                  variant="caption"
                  onClick={() => setShowAdvanced(!showAdvanced)}
                  sx={{ textDecoration: "none" }}
                >
                  {showAdvanced
                    ? "Use date picker instead"
                    : "Use relative time syntax (now-1h, now-30m, etc.)"}
                </Link>
              </Box>

              <Collapse in={showAdvanced}>
                <HelpText>
                  <Typography variant="caption" display="block" gutterBottom>
                    <strong>Relative time examples:</strong>
                  </Typography>
                  <Typography variant="caption" display="block">
                    • <code>now</code> - Current time
                  </Typography>
                  <Typography variant="caption" display="block">
                    • <code>now-5m</code> - 5 minutes ago
                  </Typography>
                  <Typography variant="caption" display="block">
                    • <code>now-1h</code> - 1 hour ago
                  </Typography>
                  <Typography variant="caption" display="block">
                    • <code>now-3d</code> - 3 days ago
                  </Typography>
                </HelpText>
              </Collapse>

              <TimezoneInfo>
                <Typography variant="body2" component="span">
                  Browser Time
                </Typography>
                <Typography variant="body2" component="span" fontWeight={600}>
                  {timezoneInfo.displayName}
                </Typography>
              </TimezoneInfo>

              <Box sx={{ display: "flex", gap: 1, justifyContent: "flex-end" }}>
                <Button variant="outlined" onClick={handleClose}>
                  Cancel
                </Button>
                <Button
                  variant="contained"
                  color="primary"
                  onClick={handleApply}
                >
                  Apply time range
                </Button>
              </Box>
            </CustomTimeRangeContainer>
          </PopoverContent>
        </Popover>
      </TimeRangePickerContainer>
    </LocalizationProvider>
  );
};
