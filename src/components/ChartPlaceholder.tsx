import { Box, styled } from "@mui/material";

export const ChartPlaceholder = styled(Box)(({ theme }) => ({
  height: 300,
  backgroundColor: theme.palette.background.paper,
  border: "1px dashed",
  borderColor: theme.palette.divider,
  borderRadius: theme.shape.borderRadius,
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  flexDirection: "column",
  gap: theme.spacing(2),
}));

export const SmallChartPlaceholder = styled(ChartPlaceholder)({
  height: 200,
});
