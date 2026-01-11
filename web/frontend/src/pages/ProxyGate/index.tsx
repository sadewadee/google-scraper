import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { proxyApi } from "@/api/proxy"
import { Button } from "@/components/UI/Button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/UI/Card"
import { Input } from "@/components/UI/Input"
import { Plus, RefreshCw, Trash2, CheckCircle2, Globe } from "lucide-react"
import { useState } from "react"

export default function ProxyGate() {
    const queryClient = useQueryClient()
    const [newSourceUrl, setNewSourceUrl] = useState("")

    const { data: stats, isLoading: statsLoading } = useQuery({
        queryKey: ["proxyStats"],
        queryFn: proxyApi.getStats,
        refetchInterval: 5000
    })

    const { data: sources, isLoading: sourcesLoading } = useQuery({
        queryKey: ["proxySources"],
        queryFn: proxyApi.getSources,
    })

    const addSourceMutation = useMutation({
        mutationFn: proxyApi.addSource,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["proxySources"] })
            setNewSourceUrl("")
        },
    })

    const deleteSourceMutation = useMutation({
        mutationFn: proxyApi.deleteSource,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["proxySources"] })
        },
    })

    const refreshMutation = useMutation({
        mutationFn: proxyApi.refresh,
        onSuccess: () => {
            // Invalidate stats to show update immediately if possible
            queryClient.invalidateQueries({ queryKey: ["proxyStats"] })
        }
    })

    // Toggle active status (assuming updateSource exists or we implement logic)
    const toggleSourceMutation = useMutation({
        mutationFn: ({ id, active }: { id: number, active: boolean }) => proxyApi.updateSource(id, active),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["proxySources"] })
        },
    })


    const handleAddSource = (e: React.FormEvent) => {
        e.preventDefault()
        if (!newSourceUrl) return
        addSourceMutation.mutate(newSourceUrl)
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <h2 className="text-3xl font-bold tracking-tight">ProxyGate Manager</h2>
                <Button
                    variant="outline"
                    onClick={() => refreshMutation.mutate()}
                    disabled={refreshMutation.isPending}
                >
                    <RefreshCw className={`mr-2 h-4 w-4 ${refreshMutation.isPending ? 'animate-spin' : ''}`} />
                    Refresh Pool
                </Button>
            </div>

            {/* Stats Overview */}
            <div className="grid gap-4 md:grid-cols-3">
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Total Proxies</CardTitle>
                        <Globe className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">{statsLoading ? "..." : stats?.data?.total_proxies || 0}</div>
                        <p className="text-xs text-muted-foreground">Buffered IPs</p>
                    </CardContent>
                </Card>
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Healthy Pool</CardTitle>
                        <CheckCircle2 className="h-4 w-4 text-green-500" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">{statsLoading ? "..." : stats?.data?.healthy_proxies || 0}</div>
                        <p className="text-xs text-muted-foreground">Ready for scraping</p>
                    </CardContent>
                </Card>
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Last Fetch</CardTitle>
                        <RefreshCw className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">
                            {statsLoading ? "..." : (stats?.data?.last_updated ? new Date(stats.data.last_updated).toLocaleString() : "Never")}
                        </div>
                        <p className="text-xs text-muted-foreground">Fetch cycle status</p>
                    </CardContent>
                </Card>
            </div>

            {/* Sources Management */}
            <div className="space-y-4">
                <div className="flex items-center justify-between">
                    <h3 className="text-xl font-semibold">Proxy Sources</h3>
                </div>

                <Card>
                    <CardContent className="p-6">
                        <form onSubmit={handleAddSource} className="flex gap-4 mb-6">
                            <Input
                                placeholder="Enter public proxy list URL (TXT/CSV)..."
                                value={newSourceUrl}
                                onChange={(e) => setNewSourceUrl(e.target.value)}
                                className="flex-1"
                            />
                            <Button type="submit" disabled={addSourceMutation.isPending}>
                                <Plus className="mr-2 h-4 w-4" />
                                Add Source
                            </Button>
                        </form>

                        <div className="rounded-md border">
                            <div className="p-4 grid grid-cols-12 gap-4 border-b bg-muted/50 font-medium text-sm">
                                <div className="col-span-8">Source URL</div>
                                <div className="col-span-2 text-center">Status</div>
                                <div className="col-span-2 text-right">Actions</div>
                            </div>
                            <div className="divide-y">
                                {sourcesLoading ? (
                                    <div className="p-4 text-center text-muted-foreground">Loading sources...</div>
                                ) : sources?.data?.length === 0 ? (
                                    <div className="p-4 text-center text-muted-foreground">No sources configured. Add one to start.</div>
                                ) : (
                                    sources?.data?.map((source) => (
                                        <div key={source.id} className="p-4 grid grid-cols-12 gap-4 items-center text-sm">
                                            <div className="col-span-8 font-mono truncate" title={source.url}>
                                                {source.url}
                                            </div>
                                            <div className="col-span-2 text-center">
                                                <Button
                                                    variant="ghost"
                                                    size="sm"
                                                    className={source.active ? "text-green-600 hover:text-green-700" : "text-gray-400"}
                                                    onClick={() => toggleSourceMutation.mutate({ id: source.id, active: !source.active })}
                                                >
                                                    {source.active ? "Active" : "Inactive"}
                                                </Button>
                                            </div>
                                            <div className="col-span-2 text-right">
                                                <Button
                                                    variant="ghost"
                                                    size="icon"
                                                    className="text-destructive hover:bg-destructive/10"
                                                    onClick={() => deleteSourceMutation.mutate(source.id)}
                                                    disabled={deleteSourceMutation.isPending}
                                                >
                                                    <Trash2 className="h-4 w-4" />
                                                </Button>
                                            </div>
                                        </div>
                                    ))
                                )}
                            </div>
                        </div>
                    </CardContent>
                </Card>
            </div>
        </div>
    )
}
