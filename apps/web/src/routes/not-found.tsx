import { Compass } from "lucide-react";
import { Link } from "react-router";
import { Button } from "@/components/ui/button";

export default function NotFoundPage() {
  return (
    <div className="grid min-h-[60vh] place-items-center">
      <div className="flex flex-col items-center gap-4 text-center">
        <Compass aria-hidden className="size-8 text-muted-foreground/60" />
        <div>
          <h1 className="text-lg font-semibold">Page not found</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            This page doesn't exist or has moved.
          </p>
        </div>
        <Button asChild variant="outline">
          <Link to="/">Back to overview</Link>
        </Button>
      </div>
    </div>
  );
}
