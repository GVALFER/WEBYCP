import { PanelsTopLeft } from "lucide-react";
import { cn } from "@/utils/classnames";

type Props = {
    className?: string;
    compact?: boolean;
};

export const Brand = ({ className, compact = false }: Props) => (
    <div className={cn("flex items-center gap-3", className)}>
        <div className="brand-mark" aria-hidden="true">
            <PanelsTopLeft className="size-5" strokeWidth={1.8} />
        </div>
        {!compact && (
            <div className="min-w-0">
                <div className="truncate text-sm font-semibold tracking-[-0.02em]">WEBYCP</div>
                <div className="mt-0.5 text-[9px] font-semibold tracking-[0.2em] text-foreground-400 uppercase">
                    Control panel
                </div>
            </div>
        )}
    </div>
);
