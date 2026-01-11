import { Link, useLocation } from "react-router-dom"
import {
    LayoutDashboard,
    ListCheck,
    Server,
    Settings,
    Menu,
    X,
    Globe
} from "lucide-react"
import { cn } from "@/lib/utils"
import { useState } from "react"
import { Button } from "../UI/Button"

const NAV_ITEMS = [
    { label: "Dashboard", href: "/", icon: LayoutDashboard },
    { label: "Jobs", href: "/jobs", icon: ListCheck },
    { label: "Workers", href: "/workers", icon: Server },
    { label: "Proxy Gate", href: "/proxies", icon: Globe },
    { label: "Settings", href: "/settings", icon: Settings },
]

export function Sidebar({ className }: { className?: string }) {
    const location = useLocation()

    return (
        <aside className={cn("w-64 bg-neu-base h-screen flex flex-col shadow-neu-flat z-20", className)}>
            <div className="p-6 border-b">
                <h1 className="text-xl font-bold bg-gradient-to-r from-primary to-blue-400 bg-clip-text text-transparent">
                    G-Scraper
                </h1>
            </div>

            <nav className="flex-1 p-4 space-y-2">
                {NAV_ITEMS.map((item) => {
                    const Icon = item.icon
                    const isActive = location.pathname === item.href

                    return (
                        <Link
                            key={item.href}
                            to={item.href}
                            className={cn(
                                "flex items-center gap-3 px-3 py-2.5 rounded-md text-sm font-medium transition-colors",
                                isActive
                                    ? "text-primary font-bold shadow-neu-pressed rounded-xl bg-neu-base"
                                    : "text-muted-foreground hover:text-primary transition-all hover:scale-105"
                            )}
                        >
                            <Icon className="h-4 w-4" />
                            {item.label}
                        </Link>
                    )
                })}
            </nav>

            <div className="p-4">
                <div className="flex items-center gap-3 px-3 py-2 bg-neu-base rounded-xl shadow-neu-pressed">
                    <div className="w-8 h-8 rounded-full bg-neu-base shadow-neu-flat flex items-center justify-center text-primary">
                        <span className="text-xs font-bold">A</span>
                    </div>
                    <div>
                        <p className="text-sm font-medium">Admin</p>
                        <p className="text-xs text-muted-foreground">Online</p>
                    </div>
                </div>
            </div>
        </aside>
    )
}

export function Header({ onMenuClick }: { onMenuClick: () => void }) {
    return (
        <header className="h-16 bg-neu-base flex items-center px-6 justify-between lg:justify-end shadow-neu-flat z-10 relative">
            <Button
                variant="ghost"
                size="icon"
                className="lg:hidden"
                onClick={onMenuClick}
            >
                <Menu className="h-5 w-5" />
            </Button>

            <div className="flex items-center gap-4">
                {/* Add user menu or notifications here if needed */}
            </div>
        </header>
    )
}

export function AppShell({ children }: { children: React.ReactNode }) {
    const [sidebarOpen, setSidebarOpen] = useState(false)

    return (
        <div className="flex min-h-screen bg-background text-foreground">
            {/* Mobile Sidebar Overlay */}
            {sidebarOpen && (
                <div
                    className="fixed inset-0 bg-black/50 z-40 lg:hidden"
                    onClick={() => setSidebarOpen(false)}
                />
            )}

            {/* Sidebar */}
            <div className={cn(
                "fixed inset-y-0 left-0 z-50 transform transition-transform duration-200 ease-in-out lg:relative lg:translate-x-0",
                sidebarOpen ? "translate-x-0" : "-translate-x-full"
            )}>
                <Sidebar />
                <Button
                    variant="ghost"
                    size="icon"
                    className="absolute top-4 right-4 lg:hidden"
                    onClick={() => setSidebarOpen(false)}
                >
                    <X className="h-5 w-5" />
                </Button>
            </div>

            {/* Main Content */}
            <div className="flex-1 flex flex-col min-w-0">
                <Header onMenuClick={() => setSidebarOpen(true)} />
                <main className="flex-1 p-6 overflow-y-auto">
                    <div className="max-w-7xl mx-auto space-y-6">
                        {children}
                    </div>
                </main>
            </div>
        </div>
    )
}
