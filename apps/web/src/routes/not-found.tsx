import { Compass } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";
import { Button } from "@/components/ui/button";

export default function NotFoundPage() {
  const { t } = useTranslation();
  return (
    <div className="grid min-h-[60vh] place-items-center">
      <div className="flex flex-col items-center gap-4 text-center">
        <Compass aria-hidden className="size-8 text-muted-foreground/60" />
        <div>
          <h1 className="text-lg font-semibold">{t("notFound.title")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {t("notFound.description")}
          </p>
        </div>
        <Button asChild variant="outline">
          <Link to="/">{t("notFound.backHome")}</Link>
        </Button>
      </div>
    </div>
  );
}
