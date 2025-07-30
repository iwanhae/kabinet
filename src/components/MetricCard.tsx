import React from "react";
import {
  Card,
  CardContent,
  Typography,
  Box,
  Chip,
  styled,
} from "@mui/material";
import TrendingUpIcon from "@mui/icons-material/TrendingUp";
import TrendingDownIcon from "@mui/icons-material/TrendingDown";

interface MetricCardProps {
  title: string;
  value: string | number;
  change?: number;
  subtitle?: string;
  onClick?: () => void;
}

const StyledCard = styled(Card, {
  shouldForwardProp: (prop) => prop !== "isClickable",
})<{ isClickable: boolean }>(({ theme, isClickable }) => ({
  height: "100%",
  cursor: isClickable ? "pointer" : "default",
  transition: "all 0.2s ease-in-out",
  "&:hover": isClickable
    ? {
        transform: "translateY(-2px)",
        boxShadow:
          theme.palette.mode === "dark"
            ? "0 8px 25px rgba(29, 155, 240, 0.15)"
            : "0 8px 25px rgba(0, 0, 0, 0.1)",
      }
    : {},
}));

const TitleTypography = styled(Typography)(({ theme }) => ({
  fontSize: "0.875rem",
  fontWeight: 500,
  marginBottom: theme.spacing(2),
}));

const ValueTypography = styled(Typography)({
  fontWeight: 700,
  fontSize: "2rem",
  lineHeight: 1.2,
});

const ChangeChip = styled(Chip, {
  shouldForwardProp: (prop) => prop !== "isPositive",
})<{ isPositive: boolean }>(({ theme, isPositive }) => ({
  backgroundColor: isPositive
    ? theme.palette.success.main
    : theme.palette.error.main,
  color: "white",
  fontWeight: 600,
  fontSize: "0.75rem",
  "& .MuiChip-icon": {
    color: "white",
    fontSize: "0.875rem",
  },
}));

const MetricCard: React.FC<MetricCardProps> = ({
  title,
  value,
  change,
  subtitle,
  onClick,
}) => {
  const isPositiveChange = change !== undefined && change > 0;

  const formatChange = (changeValue: number) => {
    const sign = changeValue > 0 ? "+" : "";
    return `${sign}${changeValue}%`;
  };

  return (
    <StyledCard isClickable={!!onClick} onClick={onClick}>
      <CardContent sx={{ p: 3 }}>
        <TitleTypography variant="body2" color="text.secondary" gutterBottom>
          {title}
        </TitleTypography>

        <Box sx={{ display: "flex", alignItems: "baseline", gap: 2, mb: 1 }}>
          <ValueTypography variant="h4">{value}</ValueTypography>

          {change !== undefined && (
            <ChangeChip
              icon={
                isPositiveChange ? <TrendingUpIcon /> : <TrendingDownIcon />
              }
              label={formatChange(change)}
              size="small"
              isPositive={isPositiveChange}
            />
          )}
        </Box>

        {subtitle && (
          <Typography
            variant="body2"
            color="text.secondary"
            sx={{ fontSize: "0.75rem" }}
          >
            {subtitle}
          </Typography>
        )}
      </CardContent>
    </StyledCard>
  );
};

export default MetricCard;
