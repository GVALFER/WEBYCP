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

type AreaTooltipProps = TooltipProps & {
    formatLabel?: ChartAreaProps["formatXAxisTick"];
    formatValue?: ChartAreaProps["formatTooltipValue"];
    unit?: string;
};

const AreaTooltip = ({
    active,
    payload,
    label,
    formatLabel,
    formatValue,
    unit,
}: AreaTooltipProps) => {
    if (!active || !payload?.length) return null;

    return (
        <div className="rounded-xl border border-divider bg-overlay/92 p-3 shadow-lg backdrop-blur-sm">
            <div className="mb-2 text-sm text-muted">
                {formatLabel && label !== undefined ? formatLabel(label) : label}
            </div>
            {payload.map((entry) => {
                const [value, name] = formatValue
                    ? formatValue(entry.value, entry.name)
                    : [unit ? `${entry.value}${unit}` : entry.value.toString(), entry.name];

                return (
                    <div key={entry.dataKey} className="flex items-center gap-2">
                        <span
                            className="size-3 rounded-full"
                            style={{ backgroundColor: entry.color }}
                        />
                        <span className="text-sm">
                            {name}: {value}
                        </span>
                    </div>
                );
            })}
        </div>
    );
};

type AreaLegendProps = {
    areas: AreaConfig[];
    hidden: Set<string>;
    onToggle: (dataKey: string) => void;
};

const AreaLegend = ({ areas, hidden, onToggle }: AreaLegendProps) => (
    <div className="mt-4 flex flex-wrap justify-center gap-4">
        {areas.map((area) => (
            <button
                key={area.dataKey}
                type="button"
                className={cn(
                    "flex items-center gap-2 rounded-lg px-3 py-1 text-sm text-muted transition-colors hover:bg-surface-secondary",
                    hidden.has(area.dataKey) && "opacity-50",
                )}
                onClick={() => onToggle(area.dataKey)}
            >
                <span
                    className="size-3 rounded-full"
                    style={{ backgroundColor: area.color }}
                />
                {area.name}
            </button>
        ))}
    </div>
);

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
                                content={
                                    <AreaTooltip
                                        formatLabel={formatXAxisTick}
                                        formatValue={formatTooltipValue}
                                        unit={yAxisUnit}
                                    />
                                }
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
            {showLegend && (
                <AreaLegend
                    areas={areas}
                    hidden={hiddenAreas}
                    onToggle={handleLegendClick}
                />
            )}
        </div>
    );
};

export default ChartArea;
