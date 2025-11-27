import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, LineChart, Line } from 'recharts';
import { Database } from 'lucide-react';

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
            <div className="overflow-x-auto rounded-lg border border-gray-800/50">
                <table className="w-full text-sm text-left text-text-secondary">
                    <thead className="text-xs text-text-tertiary uppercase bg-bg-app/50 border-b border-gray-800/50">
                        <tr>
                            {headers.map(header => (
                                <th key={header} className="px-4 py-3 font-medium">{header}</th>
                            ))}
                        </tr>
                    </thead>
                    <tbody>
                        {data.content.map((row, idx) => (
                            <tr key={idx} className="border-b border-gray-800/50 last:border-0 hover:bg-bg-panel/50 transition-colors">
                                {headers.map(header => (
                                    <td key={`${idx}-${header}`} className="px-4 py-3 font-mono text-text-primary">
                                        {String(row[header])}
                                    </td>
                                ))}
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        );
    };

    const renderBarChart = () => {
        const keys = Object.keys(data.content[0]).filter(k => k !== 'label' && k !== 'name' && k !== 'date');
        const labelKey = Object.keys(data.content[0]).find(k => k === 'label' || k === 'name' || k === 'date') || Object.keys(data.content[0])[0];

        return (
            <div className="h-64 w-full mt-4">
                <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={data.content}>
                        <CartesianGrid strokeDasharray="3 3" stroke="#374151" vertical={false} />
                        <XAxis dataKey={labelKey} stroke="#9CA3AF" fontSize={12} tickLine={false} axisLine={false} />
                        <YAxis stroke="#9CA3AF" fontSize={12} tickLine={false} axisLine={false} />
                        <Tooltip
                            contentStyle={{ backgroundColor: '#1F2937', borderColor: '#374151', color: '#F3F4F6' }}
                            itemStyle={{ color: '#F3F4F6' }}
                        />
                        <Legend />
                        {keys.map((key, index) => (
                            <Bar key={key} dataKey={key} fill={index === 0 ? "#8B5CF6" : "#3B82F6"} radius={[4, 4, 0, 0]} />
                        ))}
                    </BarChart>
                </ResponsiveContainer>
            </div>
        );
    };

    const renderLineChart = () => {
        const keys = Object.keys(data.content[0]).filter(k => k !== 'label' && k !== 'name' && k !== 'date');
        const labelKey = Object.keys(data.content[0]).find(k => k === 'label' || k === 'name' || k === 'date') || Object.keys(data.content[0])[0];

        return (
            <div className="h-64 w-full mt-4">
                <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={data.content}>
                        <CartesianGrid strokeDasharray="3 3" stroke="#374151" vertical={false} />
                        <XAxis dataKey={labelKey} stroke="#9CA3AF" fontSize={12} tickLine={false} axisLine={false} />
                        <YAxis stroke="#9CA3AF" fontSize={12} tickLine={false} axisLine={false} />
                        <Tooltip
                            contentStyle={{ backgroundColor: '#1F2937', borderColor: '#374151', color: '#F3F4F6' }}
                            itemStyle={{ color: '#F3F4F6' }}
                        />
                        <Legend />
                        {keys.map((key, index) => (
                            <Line key={key} type="monotone" dataKey={key} stroke={index === 0 ? "#8B5CF6" : "#3B82F6"} strokeWidth={2} dot={{ r: 4 }} />
                        ))}
                    </LineChart>
                </ResponsiveContainer>
            </div>
        );
    };

    return (
        <div className="mt-4 p-4 rounded-xl bg-bg-panel border border-gray-800/50 shadow-sm">
            <div className="flex items-center gap-2 mb-4">
                <Database className="w-4 h-4 text-accent-primary" />
                <h3 className="text-sm font-semibold text-text-primary uppercase tracking-wide">{data.title}</h3>
            </div>

            {data.type === 'table' && renderTable()}
            {data.type === 'bar_chart' && renderBarChart()}
            {data.type === 'line_chart' && renderLineChart()}
        </div>
    );
};
