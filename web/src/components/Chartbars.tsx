"use client";

import { useTheme } from "@/lib/theme";
import { cn } from "@/utils/classnames";
import { Spinner } from "@heroui/react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

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

type BarConfig = {
    dataKey: string;
    color: string;
    name: string;
    radius?: [number, number, number, number];
    stackId?: string;
};

type ChartBarsProps = {
    className?: string;
    data: ChartDataPoint[];
    bars: BarConfig[];
    xAxisDataKey?: string;
    yAxisUnit?: string;
    showGrid?: boolean;
    showLegend?: boolean;
    showTooltip?: boolean;
    height?: number;
    margin?: ChartMargin;
    formatXAxisTick?: (value: string | number) => string;
    formatYAxisTick?: (value: number) => string;
    formatTooltipValue?: (value: number, name: string) => [string, string];
    formatTooltipLabel?: (label: string | number) => string;
    maxBarSize?: number;
};

const ChartBars = ({
    className,
    data = [],
    bars = [],
    xAxisDataKey = "name",
    yAxisUnit,
    showGrid = true,
    showLegend = true,
    showTooltip = true,
    height = 288,
    margin = { top: 10, right: 30, left: 0, bottom: 0 },
    formatXAxisTick,
    formatYAxisTick,
    formatTooltipValue,
    formatTooltipLabel,
    maxBarSize = 80,
}: ChartBarsProps) => {
    const [isClient, setIsClient] = useState(false);
    const [hiddenBars, setHiddenBars] = useState<Set<string>>(new Set());
    const { theme } = useTheme();

    const toggleBar = useCallback((dataKey: string) => {
        setHiddenBars((prev) => {
            const newSet = new Set(prev);
            if (newSet.has(dataKey)) {
                newSet.delete(dataKey);
            } else {
                newSet.add(dataKey);
            }
            return newSet;
        });
    }, []);

    const visibleBars = useMemo(() => {
        return bars.filter((bar) => !hiddenBars.has(bar.dataKey));
    }, [bars, hiddenBars]);

    const handleLegendClick = useCallback(
        (dataKey: string) => {
            toggleBar(dataKey);
        },
        [toggleBar],
    );

    useEffect(() => {
        setIsClient(true);
    }, []);

    const CustomTooltip = useCallback(
        ({ active, payload, label }: TooltipProps) => {
            if (!active || !payload || !payload.length) {
                return null;
            }

            return (
                <div className="bg-surface/70 rounded p-3 shadow-lg backdrop-blur-sm">
                    <p className="text-muted text-sm mb-2">
                        {formatTooltipLabel && label !== undefined
                            ? formatTooltipLabel(label)
                            : label}
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
                                    className="w-3 h-3 rounded-sm"
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
        [formatTooltipValue, formatTooltipLabel, yAxisUnit],
    );

    const CustomLegend = useCallback(() => {
        if (!showLegend) return null;

        return (
            <div className="flex flex-wrap justify-center gap-4 mt-4">
                {bars.map((bar) => {
                    const isHidden = hiddenBars.has(bar.dataKey);
                    return (
                        <button
                            key={bar.dataKey}
                            onClick={() => handleLegendClick(bar.dataKey)}
                            className={cn(
                                "flex items-center gap-2 px-3 py-1 rounded-md transition-all hover:bg-gray-800/50",
                                isHidden && "opacity-50",
                            )}
                        >
                            <div
                                className="w-3 h-3 rounded-sm"
                                style={{ backgroundColor: bar.color }}
                            />
                            <span className="text-muted text-sm">{bar.name}</span>
                        </button>
                    );
                })}
            </div>
        );
    }, [bars, hiddenBars, handleLegendClick, showLegend]);

    if (!data || data.length === 0) {
        return (
            <div
                className={cn(
                    "flex items-center justify-center h-72 bg-surface-secondary/50 rounded",
                    className,
                )}
            >
                <div className="text-muted text-sm">No data available</div>
            </div>
        );
    }

    if (!isClient) {
        return (
            <div
                className="flex items-center justify-center rounded animate-pulse"
                style={{ height: height }}
            >
                <Spinner />
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
            <div style={{ height: height || 288 }}>
                <ResponsiveContainer
                    width="100%"
                    height="100%"
                    initialDimension={{ width: 320, height: 200 }}
                    style={{ outline: "none" }}
                >
                    <BarChart
                        data={data}
                        margin={margin}
                        style={{ outline: "none" }}
                        barCategoryGap="20%"
                        barGap={4}
                    >
                        <defs>
                            {visibleBars.map((bar) => (
                                <linearGradient
                                    key={`barGradient-${bar.dataKey}`}
                                    id={`barGradient-${bar.dataKey}`}
                                    x1="0%"
                                    y1="0%"
                                    x2="0%"
                                    y2="100%"
                                >
                                    <stop offset="0%" stopColor={bar.color} stopOpacity={1} />
                                    <stop offset="100%" stopColor={bar.color} stopOpacity={0.5} />
                                </linearGradient>
                            ))}
                        </defs>
                        {showGrid && (
                            <CartesianGrid
                                strokeDasharray="3 3"
                                stroke={theme === "dark" ? "#6CA3AF" : "#131315"}
                                strokeOpacity={0.2}
                                horizontal={true}
                                vertical={false}
                            />
                        )}

                        <XAxis
                            dataKey={xAxisDataKey}
                            axisLine={false}
                            tickLine={false}
                            tick={{
                                fill: theme === "dark" ? "#6CA3AF" : "#131315",
                                fontSize: 12,
                            }}
                            tickFormatter={formatXAxisTick}
                            angle={-45}
                            textAnchor="end"
                            height={60}
                            interval={0}
                        />

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

                        {showTooltip && (
                            <Tooltip
                                content={<CustomTooltip />}
                                cursor={{
                                    fill:
                                        theme === "dark"
                                            ? "rgba(255, 255, 255, 0.05)"
                                            : "rgba(0, 0, 0, 0.05)",
                                }}
                            />
                        )}

                        {visibleBars.map((bar) => (
                            <Bar
                                key={bar.dataKey}
                                dataKey={bar.dataKey}
                                stackId={bar.stackId}
                                fill={`url(#barGradient-${bar.dataKey})`}
                                radius={bar.radius || [4, 4, 0, 0]}
                                maxBarSize={maxBarSize}
                            />
                        ))}
                    </BarChart>
                </ResponsiveContainer>
            </div>
            <CustomLegend />
        </div>
    );
};

export default ChartBars;
