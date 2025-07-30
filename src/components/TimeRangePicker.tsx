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
} from "@mui/material";
import { AccessTime, ArrowDropDown } from "@mui/icons-material";
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
  padding: "8px",
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

const AbsoluteTimeRangeContainer = styled(Box)(({ theme }) => ({
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
  const { from, to } = useTimeRangeStore();
  const setTimeRangeWithUrl = useTimeRangeFromUrl();

  const [tempFrom, setTempFrom] = useState(from);
  const [tempTo, setTempTo] = useState(to);
  const [tempFromDate, setTempFromDate] = useState<Dayjs | null>(
    parseTimeString(from),
  );
  const [tempToDate, setTempToDate] = useState<Dayjs | null>(
    parseTimeString(to),
  );
  const [isAbsoluteMode, setIsAbsoluteMode] = useState(
    !isRelativeTime(from) || !isRelativeTime(to),
  );

  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    setAnchorEl(event.currentTarget);
    setTempFrom(from);
    setTempTo(to);
    setTempFromDate(parseTimeString(from));
    setTempToDate(parseTimeString(to));
    setIsAbsoluteMode(!isRelativeTime(from) || !isRelativeTime(to));
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
      const timeString = formatTimeString(date);
      if (field === "from") {
        setTempFromDate(date);
        setTempFrom(timeString);
      } else {
        setTempToDate(date);
        setTempTo(timeString);
      }
    }
  };

  const handleApply = () => {
    if (isAbsoluteMode && tempFromDate && tempToDate) {
      setTimeRangeWithUrl(
        formatTimeString(tempFromDate),
        formatTimeString(tempToDate),
      );
    } else {
      setTimeRangeWithUrl(tempFrom, tempTo);
    }
    handleClose();
  };

  const open = Boolean(anchorEl);
  const id = open ? "time-range-popover" : undefined;
  const timezoneInfo = getCurrentTimezone();

  const displayLabel =
    quickRanges.find((r) => r.from === from && r.to === "now")?.label ||
    `${from} to ${to}`;

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
                      selected={from === range.from && to === range.to}
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
                            from === range.from && to === range.to ? 600 : 400,
                        }}
                      />
                    </ListItemButton>
                  ))}
                </List>
              </QuickRangesList>
            </QuickRangesContainer>

            <AbsoluteTimeRangeContainer>
              <Typography variant="h6" gutterBottom fontWeight={600}>
                Absolute time range
              </Typography>

              <Box
                sx={{ display: "flex", alignItems: "center", gap: 1, mb: 2 }}
              >
                <Button
                  variant={!isAbsoluteMode ? "contained" : "outlined"}
                  size="small"
                  onClick={() => setIsAbsoluteMode(false)}
                >
                  Relative
                </Button>
                <Button
                  variant={isAbsoluteMode ? "contained" : "outlined"}
                  size="small"
                  onClick={() => setIsAbsoluteMode(true)}
                >
                  Absolute
                </Button>
              </Box>

              {isAbsoluteMode ? (
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
                    value={tempFrom}
                    onChange={(e) => setTempFrom(e.target.value)}
                    fullWidth
                    size="small"
                    placeholder="e.g., now-1h, now-30m"
                  />
                  <TextField
                    label="To"
                    value={tempTo}
                    onChange={(e) => setTempTo(e.target.value)}
                    fullWidth
                    size="small"
                    placeholder="e.g., now"
                  />
                </>
              )}

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
            </AbsoluteTimeRangeContainer>
          </PopoverContent>
        </Popover>
      </TimeRangePickerContainer>
    </LocalizationProvider>
  );
};
