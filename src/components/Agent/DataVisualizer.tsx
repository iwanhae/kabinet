import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, LineChart, Line } from 'recharts';
import {
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Paper,
    Typography,
    Box,
    Card,
    CardContent
} from '@mui/material';
import StorageIcon from '@mui/icons-material/Storage';

interface DataVisualizerProps {
    data: {
        type: 'table' | 'bar_chart' | 'line_chart';
        title: string;
        content: any[];
    };
}

export const DataVisualizer: React.FC<DataVisualizerProps> = ({ data }) => {
    if (!data.content || data.content.length === 0) return null;

    const renderTable = () => {
        const headers = Object.keys(data.content[0]);
        return (
            <TableContainer component={Paper} variant="outlined" sx={{ mt: 2, maxHeight: 400 }}>
                <Table size="small" stickyHeader>
                    <TableHead>
                        <TableRow>
                            {headers.map(header => (
                                <TableCell key={header} sx={{ fontWeight: 'bold', textTransform: 'uppercase' }}>
                                    {header}
                                </TableCell>
                            ))}
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {data.content.map((row, idx) => (
                            <TableRow key={idx} hover>
                                {headers.map(header => (
                                    <TableCell key={`${idx}-${header}`} sx={{ fontFamily: 'monospace' }}>
                                        {String(row[header])}
                                    </TableCell>
                                ))}
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>
        );
    };

    const renderBarChart = () => {
        const keys = Object.keys(data.content[0]).filter(k => k !== 'label' && k !== 'name' && k !== 'date');
        const labelKey = Object.keys(data.content[0]).find(k => k === 'label' || k === 'name' || k === 'date') || Object.keys(data.content[0])[0];

        return (
            <Box sx={{ height: 300, width: '100%', mt: 2 }}>
                <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={data.content}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey={labelKey} />
                        <YAxis />
                        <Tooltip
                            contentStyle={{ backgroundColor: '#1F2937', color: '#F3F4F6', border: 'none' }}
                            itemStyle={{ color: '#F3F4F6' }}
                        />
                        <Legend />
                        {keys.map((key, index) => (
                            <Bar key={key} dataKey={key} fill={index === 0 ? "#8B5CF6" : "#3B82F6"} />
                        ))}
                    </BarChart>
                </ResponsiveContainer>
            </Box>
        );
    };

    const renderLineChart = () => {
        const keys = Object.keys(data.content[0]).filter(k => k !== 'label' && k !== 'name' && k !== 'date');
        const labelKey = Object.keys(data.content[0]).find(k => k === 'label' || k === 'name' || k === 'date') || Object.keys(data.content[0])[0];

        return (
            <Box sx={{ height: 300, width: '100%', mt: 2 }}>
                <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={data.content}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey={labelKey} />
                        <YAxis />
                        <Tooltip
                            contentStyle={{ backgroundColor: '#1F2937', color: '#F3F4F6', border: 'none' }}
                            itemStyle={{ color: '#F3F4F6' }}
                        />
                        <Legend />
                        {keys.map((key, index) => (
                            <Line key={key} type="monotone" dataKey={key} stroke={index === 0 ? "#8B5CF6" : "#3B82F6"} strokeWidth={2} />
                        ))}
                    </LineChart>
                </ResponsiveContainer>
            </Box>
        );
    };

    return (
        <Card variant="outlined" sx={{ mt: 2, mb: 2 }}>
            <CardContent>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                    <StorageIcon color="primary" fontSize="small" />
                    <Typography variant="subtitle2" sx={{ textTransform: 'uppercase', fontWeight: 'bold' }}>
                        {data.title}
                    </Typography>
                </Box>

                {data.type === 'table' && renderTable()}
                {data.type === 'bar_chart' && renderBarChart()}
                {data.type === 'line_chart' && renderLineChart()}
            </CardContent>
        </Card>
    );
};
