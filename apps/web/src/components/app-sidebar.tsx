import { BookOpenCheck, BookText, Building2, GitCompareArrows, Handshake, LayoutDashboard, ListChecks, PlugZap, Scale, ScrollText, ShieldCheck } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link, useLocation } from "react-router";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem
} from "@/components/ui/sidebar";

const items = [
  { key: "overview", url: "/", icon: LayoutDashboard },
  { key: "transactions", url: "/transactions", icon: ListChecks },
  { key: "validation", url: "/validation", icon: ShieldCheck },
  { key: "accounts", url: "/accounts", icon: BookOpenCheck },
  { key: "journal", url: "/journal", icon: BookText },
  { key: "commitments", url: "/commitments", icon: Handshake },
  { key: "balances", url: "/balances", icon: Scale },
  { key: "ingestion", url: "/ingestion", icon: PlugZap },
  { key: "reconciliation", url: "/reconciliation", icon: GitCompareArrows },
  { key: "institutions", url: "/institutions", icon: Building2 },
  { key: "audit", url: "/audit", icon: ScrollText }
];

export function AppSidebar() {
  const location = useLocation();
  const { t } = useTranslation();

  return (
    <Sidebar collapsible="icon">
      <SidebarHeader>
        <div className="flex items-center gap-2.5 px-2 py-1.5">
          <div className="grid size-7 shrink-0 place-items-center rounded-lg bg-primary text-xs font-bold text-primary-foreground">
            OT
          </div>
          <span className="truncate font-semibold group-data-[collapsible=icon]:hidden">{t("nav.product")}</span>
        </div>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <nav aria-label={t("nav.primary")}>
              <SidebarMenu>
                {items.map((item) => (
                  <SidebarMenuItem key={item.url}>
                    <SidebarMenuButton
                      asChild
                      isActive={location.pathname === item.url}
                      tooltip={t(`nav.${item.key}`)}
                    >
                      <Link to={item.url}>
                        <item.icon aria-hidden />
                        <span>{t(`nav.${item.key}`)}</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </nav>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  );
}
