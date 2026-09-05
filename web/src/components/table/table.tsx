"use client";

import { Spinner, Table as HeroTable } from "@heroui/react";
import type { ReactNode } from "react";
import type { Pagination } from "@/contracts/types";
import { Paginate } from "./paginate";
import type { TableState } from "./useTable";

export type TableColumn<T> = {
    id: string;
    label: string;
    isRowHeader?: boolean;
    headerClassName?: string;
    cellClassName?: string;
    render: (item: T) => ReactNode;
};

type Props<T extends object> = {
    columns: TableColumn<T>[];
    data?: TableData<T>;
    pending: boolean;
    getKey?: (item: T) => string;
    table: TableState;
};

type TableData<T> = {
    items: T[];
    pagination: Pagination;
};

export const Table = <T extends object,>({ columns, data, pending, getKey, table }: Props<T>) => {
    const itemKey = (item: T) => getKey?.(item) ?? String((item as { id: string }).id);
    const tableData =
        data ??
        ({
            items: [],
            pagination: {
                page: table.page.page,
                size: table.page.size,
                totalItems: 0,
                totalPages: 0,
            },
        } satisfies TableData<T>);

    return (
        <HeroTable className="data-table" variant="secondary">
            <HeroTable.ScrollContainer>
                <HeroTable.Content aria-label="Data table" aria-busy={pending}>
                    <HeroTable.Header>
                        {columns.map((column) => (
                            <HeroTable.Column
                                key={column.id}
                                id={column.id}
                                className={column.headerClassName}
                                isRowHeader={column.isRowHeader}
                            >
                                {column.label}
                            </HeroTable.Column>
                        ))}
                    </HeroTable.Header>
                    <HeroTable.Body
                        renderEmptyState={() => (
                            <div className="flex min-h-32 items-center justify-center px-6 py-12 text-sm text-foreground-400">
                                {data ? "No records found." : <Spinner size="sm" />}
                            </div>
                        )}
                    >
                        {tableData.items.map((item) => {
                            const key = itemKey(item);
                            return (
                                <HeroTable.Row key={key} id={key}>
                                    {columns.map((column) => (
                                        <HeroTable.Cell
                                            key={column.id}
                                            className={column.cellClassName}
                                        >
                                            {column.render(item)}
                                        </HeroTable.Cell>
                                    ))}
                                </HeroTable.Row>
                            );
                        })}
                    </HeroTable.Body>
                </HeroTable.Content>
            </HeroTable.ScrollContainer>
            <HeroTable.Footer className="px-0 py-4">
                <Paginate
                    pagination={tableData.pagination}
                    page={table.page.page}
                    pending={pending}
                    onPageChange={table.setPage}
                    onSizeChange={table.setSize}
                />
            </HeroTable.Footer>
        </HeroTable>
    );
};
