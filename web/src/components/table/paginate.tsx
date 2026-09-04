"use client";

import { Pagination as HeroPagination } from "@heroui/react";
import { useEffect } from "react";
import type { Pagination } from "@/contracts/types";
import { PAGE_SIZES, pageItems } from "@/utils/pagination";

type Props = {
    pagination: Pagination;
    page: number;
    onPageChange: (page: number) => void;
    onSizeChange: (size: number) => void;
};

export const Paginate = ({ pagination, page, onPageChange, onSizeChange }: Props) => {
    const { page: currentPage, size, totalItems, totalPages } = pagination;
    const first = totalItems ? (currentPage - 1) * size + 1 : 0;
    const last = Math.min(currentPage * size, totalItems);

    useEffect(() => {
        if (page !== currentPage) onPageChange(currentPage);
    }, [currentPage, onPageChange, page]);

    return (
        <HeroPagination className="border-t border-divider px-5 py-4" size="sm">
            <HeroPagination.Summary className="flex-wrap">
                <span>
                    Showing {first}–{last} of {totalItems}
                </span>
                <label className="flex items-center gap-2">
                    <span>Rows</span>
                    <select
                        className="h-8 rounded-lg border border-border bg-field-background px-2 text-foreground outline-none focus:border-accent"
                        aria-label="Rows per page"
                        value={size}
                        onChange={(event) => onSizeChange(Number(event.currentTarget.value))}
                    >
                        {PAGE_SIZES.map((value) => (
                            <option key={value} value={value}>
                                {value}
                            </option>
                        ))}
                    </select>
                </label>
            </HeroPagination.Summary>

            {totalPages > 1 && (
                <HeroPagination.Content>
                    <HeroPagination.Item>
                        <HeroPagination.Previous
                            aria-label="Previous page"
                            isDisabled={currentPage <= 1}
                            onPress={() => onPageChange(currentPage - 1)}
                        >
                            <HeroPagination.PreviousIcon />
                            <span className="hidden sm:inline">Previous</span>
                        </HeroPagination.Previous>
                    </HeroPagination.Item>

                    {pageItems(currentPage, totalPages).map((item) =>
                        typeof item === "number" ? (
                            <HeroPagination.Item key={item}>
                                <HeroPagination.Link
                                    isActive={item === currentPage}
                                    aria-label={`Page ${item}`}
                                    onPress={() => onPageChange(item)}
                                >
                                    {item}
                                </HeroPagination.Link>
                            </HeroPagination.Item>
                        ) : (
                            <HeroPagination.Item key={item}>
                                <HeroPagination.Ellipsis />
                            </HeroPagination.Item>
                        ),
                    )}

                    <HeroPagination.Item>
                        <HeroPagination.Next
                            aria-label="Next page"
                            isDisabled={currentPage >= totalPages}
                            onPress={() => onPageChange(currentPage + 1)}
                        >
                            <span className="hidden sm:inline">Next</span>
                            <HeroPagination.NextIcon />
                        </HeroPagination.Next>
                    </HeroPagination.Item>
                </HeroPagination.Content>
            )}
        </HeroPagination>
    );
};
