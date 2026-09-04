"use client";

import { useTheme } from "@/lib/theme";
import { cn } from "@/utils/classnames";
import { useCallback, useMemo, useState } from "react";
import {
    Area,
    AreaChart,
    CartesianGrid,
    ResponsiveContainer,
    Tooltip,
    XAxis,
    YAxis,
} from "recharts";

// Interfaces para tipagem forte
interface ChartDataPoint {
    [key: string]: string | number | null | undefined;
}

interface ChartMargin {
    top?: number;
    right?: number;
    bottom?: number;
    left?: number;
}

interface TooltipPayloadEntry {
    dataKey: string;
    value: number;
    name: string;
    color: string;
    payload: ChartDataPoint;
}

interface TooltipProps {
    active?: boolean;
    payload?: TooltipPayloadEntry[];
    label?: string | number;
}

type AreaConfig = {
    dataKey: string;
    color: string;
    name: string;
    strokeWidth?: number;
    fillOpacity?: number;
    stackId?: string;
};

type ChartAreaProps = {
    className?: string;
    data: ChartDataPoint[];
    areas: AreaConfig[];
    xAxisDataKey?: string;
    yAxisUnit?: string;
    showGrid?: boolean;
    showLegend?: boolean;
    showTooltip?: boolean;
    showYAxis?: boolean;
    showXAxis?: boolean;
    height?: number;
    margin?: ChartMargin;
    formatXAxisTick?: (value: string | number) => string;
    formatYAxisTick?: (value: number) => string;
    formatTooltipValue?: (value: number, name: string) => [string, string];
};

const ChartArea = ({
    className,
    data = [],
    areas = [],
    xAxisDataKey = "timestamp",
    yAxisUnit,
    showGrid = true,
    showLegend = true,
    showTooltip = true,
    showYAxis = true,
    showXAxis = true,
    height = 288,
    margin = { top: 10, right: 30, left: 0, bottom: 0 },
    formatXAxisTick,
    formatYAxisTick,
    formatTooltipValue,
}: ChartAreaProps) => {
    const [hiddenAreas, setHiddenAreas] = useState<Set<string>>(new Set());
    const { theme } = useTheme();

    const toggleArea = useCallback((dataKey: string) => {
        setHiddenAreas((prev) => {
            const newSet = new Set(prev);
            if (newSet.has(dataKey)) {
                newSet.delete(dataKey);
            } else {
                newSet.add(dataKey);
            }
            return newSet;
        });
    }, []);

    const visibleAreas = useMemo(() => {
        return areas.filter((area) => !hiddenAreas.has(area.dataKey));
    }, [areas, hiddenAreas]);

    const handleLegendClick = useCallback(
        (dataKey: string) => {
            toggleArea(dataKey);
        },
        [toggleArea],
    );

    const CustomTooltip = useCallback(
        ({ active, payload, label }: TooltipProps) => {
            if (!active || !payload || !payload.length) {
                return null;
            }

            return (
                <div className="bg-surface/70 rounded p-3 shadow-lg backdrop-blur-sm">
                    <p className="text-muted text-sm mb-2">
                        {formatXAxisTick && label !== undefined ? formatXAxisTick(label) : label}
                    </p>
                    {payload.map((entry: TooltipPayloadEntry, index: number) => {
                        const [formattedValue, formattedName] = formatTooltipValue
                            ? formatTooltipValue(entry.value, entry.name)
                            : [
                                  yAxisUnit ? `${entry.value}${yAxisUnit}` : entry.value.toString(),
                                  entry.name,
                              ];

                        return (
                            <div key={index} className="flex items-center gap-2">
                                <div
                                    className="w-3 h-3 rounded-full"
                                    style={{ backgroundColor: entry.color }}
                                />
                                <span className="text-sm">
                                    {formattedName}: {formattedValue}
                                </span>
                            </div>
                        );
                    })}
                </div>
            );
        },
        [formatXAxisTick, formatTooltipValue, yAxisUnit],
    );

    const CustomLegend = useCallback(() => {
        if (!showLegend) return null;

        return (
            <div className="flex flex-wrap justify-center gap-4 mt-4">
                {areas.map((area) => {
                    const isHidden = hiddenAreas.has(area.dataKey);
                    return (
                        <button
                            key={area.dataKey}
                            onClick={() => handleLegendClick(area.dataKey)}
                            className={cn(
                                "flex items-center gap-2 px-3 py-1 rounded-md transition-all hover:bg-gray-800/50",
                                isHidden && "opacity-50",
                            )}
                        >
                            <div
                                className="w-3 h-3 rounded-full"
                                style={{ backgroundColor: area.color }}
                            />
                            <span className="text-muted text-sm">{area.name}</span>
                        </button>
                    );
                })}
            </div>
        );
    }, [areas, hiddenAreas, handleLegendClick, showLegend]);

    if (!data || data.length === 0) {
        return (
            <div
                className={cn(
                    "flex items-center justify-center bg-surface-secondary/40 rounded",
                    className,
                )}
                style={{ height }}
            >
                <div className="text-muted text-sm">No data available</div>
            </div>
        );
    }

    return (
        <div
            className={cn(
                "w-full rounded p-4 outline-none focus:outline-none [&_svg]:outline-none [&_svg]:focus:outline-none [&_svg_*]:outline-none [&_svg_*]:focus:outline-none",
                className,
            )}
        >
            <div style={{ height: height }}>
                <ResponsiveContainer
                    width="100%"
                    height="100%"
                    initialDimension={{ width: 320, height: 200 }}
                    style={{ outline: "none" }}
                >
                    <AreaChart data={data} margin={margin} style={{ outline: "none" }}>
                        <defs>
                            {visibleAreas.map((area) => (
                                <linearGradient
                                    key={area.dataKey}
                                    id={`gradient-${area.dataKey}`}
                                    x1="0"
                                    y1="0"
                                    x2="0"
                                    y2="1"
                                >
                                    <stop
                                        offset="5%"
                                        stopColor={area.color}
                                        stopOpacity={area.fillOpacity || 0.8}
                                    />
                                    <stop offset="95%" stopColor={area.color} stopOpacity={0.1} />
                                </linearGradient>
                            ))}
                        </defs>

                        {showGrid && (
                            <CartesianGrid
                                strokeDasharray="3 3"
                                stroke={theme === "dark" ? "#6CA3AF" : "#131315"}
                                strokeOpacity={0.2}
                                horizontal={true}
                                vertical={true}
                            />
                        )}

                        {showXAxis && (
                            <XAxis
                                dataKey={xAxisDataKey}
                                axisLine={false}
                                tickLine={false}
                                tick={{
                                    fill: theme === "dark" ? "#6CA3AF" : "#131315",
                                    fontSize: 12,
                                }}
                                tickFormatter={formatXAxisTick}
                            />
                        )}

                        {showYAxis && (
                            <YAxis
                                axisLine={false}
                                tickLine={false}
                                tick={{
                                    fill: theme === "dark" ? "#6CA3AF" : "#131315",
                                    fontSize: 12,
                                }}
                                tickFormatter={
                                    formatYAxisTick ||
                                    (yAxisUnit ? (value) => `${value}${yAxisUnit}` : undefined)
                                }
                            />
                        )}

                        {showTooltip && (
                            <Tooltip
                                content={<CustomTooltip />}
                                cursor={{
                                    stroke: "#6B7280",
                                    strokeWidth: 1,
                                    strokeDasharray: "5 5",
                                }}
                            />
                        )}

                        {visibleAreas.map((area) => (
                            <Area
                                key={area.dataKey}
                                type="monotone"
                                dataKey={area.dataKey}
                                stackId={area.stackId}
                                stroke={area.color}
                                strokeWidth={area.strokeWidth || 2}
                                fill={`url(#gradient-${area.dataKey})`}
                                fillOpacity={1}
                                connectNulls={false}
                                dot={false}
                                activeDot={{
                                    r: 4,
                                    fill: area.color,
                                    stroke: "#1F2937",
                                    strokeWidth: 2,
                                }}
                            />
                        ))}
                    </AreaChart>
                </ResponsiveContainer>
            </div>
            <CustomLegend />
        </div>
    );
};

export default ChartArea;
