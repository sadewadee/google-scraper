import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Link } from "react-router-dom"
import { toast } from "sonner"
import { jobsApi } from "@/api/jobs"
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/UI/Table"
import { Badge } from "@/components/UI/Badge"
import { Button } from "@/components/UI/Button"
import { Input } from "@/components/UI/Input"
import {
    Trash2,
    XCircle,
    Search,
    Eye,
    Pause,
    Play,
    Copy,
    RotateCcw
} from "lucide-react"
import { useNavigate } from "react-router-dom"

export function JobTable() {
    const [searchTerm, setSearchTerm] = useState("")
    const [selectedJobs, setSelectedJobs] = useState<Set<string>>(new Set())
    const queryClient = useQueryClient()
    const navigate = useNavigate()

    const { data: response, isLoading, error } = useQuery({
        queryKey: ["jobs"],
        queryFn: jobsApi.getAll,
        refetchInterval: (query) => {
            const jobs = query.state.data?.data
            const hasActiveJobs = jobs?.some(j => j.status === 'running' || j.status === 'pending')
            return hasActiveJobs ? 5000 : 30000
        }
    })

    const deleteMutation = useMutation({
        mutationFn: jobsApi.delete,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["jobs"] })
            toast.success("Job deleted successfully")
        },
        onError: () => {
            toast.error("Failed to delete job")
        }
    })

    const cancelMutation = useMutation({
        mutationFn: jobsApi.cancel,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["jobs"] })
            toast.success("Job cancelled")
        },
        onError: () => {
            toast.error("Failed to cancel job")
        }
    })

    const pauseMutation = useMutation({
        mutationFn: jobsApi.pause,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["jobs"] })
            toast.success("Job paused")
        },
        onError: () => {
            toast.error("Failed to pause job")
        }
    })

    const resumeMutation = useMutation({
        mutationFn: jobsApi.resume,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["jobs"] })
            toast.success("Job resumed")
        },
        onError: () => {
            toast.error("Failed to resume job")
        }
    })

    const jobs = response?.data || []
    const filteredJobs = jobs.filter(job =>
        job.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        job.config.keywords.some(k => k.toLowerCase().includes(searchTerm.toLowerCase()))
    )

    const toggleJobSelection = (jobId: string) => {
        setSelectedJobs(prev => {
            const newSet = new Set(prev)
            if (newSet.has(jobId)) {
                newSet.delete(jobId)
            } else {
                newSet.add(jobId)
            }
            return newSet
        })
    }

    const toggleAllJobs = () => {
        if (selectedJobs.size === filteredJobs.length) {
            setSelectedJobs(new Set())
        } else {
            setSelectedJobs(new Set(filteredJobs.map(j => j.id)))
        }
    }

    const handleBulkDelete = async () => {
        if (!confirm(`Are you sure you want to delete ${selectedJobs.size} jobs?`)) return
        try {
            await Promise.all([...selectedJobs].map(id => jobsApi.delete(id)))
            toast.success(`${selectedJobs.size} jobs deleted`)
            setSelectedJobs(new Set())
            queryClient.invalidateQueries({ queryKey: ["jobs"] })
        } catch {
            toast.error("Failed to delete some jobs")
        }
    }

    const handleBulkCancel = async () => {
        if (!confirm(`Are you sure you want to cancel ${selectedJobs.size} jobs?`)) return
        try {
            await Promise.all([...selectedJobs].map(id => jobsApi.cancel(id)))
            toast.success(`${selectedJobs.size} jobs cancelled`)
            setSelectedJobs(new Set())
            queryClient.invalidateQueries({ queryKey: ["jobs"] })
        } catch {
            toast.error("Failed to cancel some jobs")
        }
    }

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-2 max-w-sm">
                    <Search className="h-4 w-4 text-muted-foreground" />
                    <Input
                        placeholder="Search jobs..."
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        className="max-w-sm"
                    />
                </div>
                {selectedJobs.size > 0 && (
                    <div className="flex items-center gap-2">
                        <span className="text-sm text-muted-foreground">
                            {selectedJobs.size} selected
                        </span>
                        <Button variant="outline" size="sm" onClick={handleBulkCancel}>
                            <XCircle className="h-4 w-4 mr-1" />
                            Cancel
                        </Button>
                        <Button variant="destructive" size="sm" onClick={handleBulkDelete}>
                            <Trash2 className="h-4 w-4 mr-1" />
                            Delete
                        </Button>
                    </div>
                )}
            </div>

            {error && (
                <div className="text-destructive text-sm p-4 bg-destructive/10 rounded-md">
                    Failed to load jobs: {error instanceof Error ? error.message : "Unknown error"}
                </div>
            )}

            <div className="rounded-md bg-neu-base shadow-neu-flat">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead className="w-[50px]">
                                <input
                                    type="checkbox"
                                    checked={filteredJobs.length > 0 && selectedJobs.size === filteredJobs.length}
                                    onChange={toggleAllJobs}
                                    className="h-4 w-4 rounded border-gray-300"
                                />
                            </TableHead>
                            <TableHead>Name</TableHead>
                            <TableHead>Keywords</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Priority</TableHead>
                            <TableHead className="text-right">Progress</TableHead>
                            <TableHead>Created At</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {isLoading ? (
                            <TableRow>
                                <TableCell colSpan={8} className="h-24 text-center">
                                    Loading jobs...
                                </TableCell>
                            </TableRow>
                        ) : filteredJobs.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={8} className="h-24 text-center">
                                    No jobs found.
                                </TableCell>
                            </TableRow>
                        ) : (
                            filteredJobs.map((job) => (
                                <TableRow key={job.id}>
                                    <TableCell>
                                        <input
                                            type="checkbox"
                                            checked={selectedJobs.has(job.id)}
                                            onChange={() => toggleJobSelection(job.id)}
                                            className="h-4 w-4 rounded border-gray-300"
                                        />
                                    </TableCell>
                                    <TableCell className="font-medium">
                                        <Link to={`/jobs/${job.id}`} className="hover:underline">
                                            {job.name}
                                        </Link>
                                    </TableCell>
                                    <TableCell className="text-muted-foreground text-xs max-w-[200px] truncate">
                                        {job.config.keywords.join(", ")}
                                    </TableCell>
                                    <TableCell>
                                        <Badge variant={job.status}>{job.status}</Badge>
                                    </TableCell>
                                    <TableCell>
                                        <span className={`text-xs ${job.priority >= 8 ? 'text-red-500 font-bold' :
                                            job.priority <= 3 ? 'text-gray-500' : 'text-foreground'
                                            }`}>
                                            {job.priority}
                                        </span>
                                    </TableCell>
                                    <TableCell className="text-right">
                                        {job.progress.scraped_places}/{job.progress.total_places || "-"}
                                        {job.progress.percentage > 0 && (
                                            <span className="text-muted-foreground ml-1">
                                                ({Math.round(job.progress.percentage)}%)
                                            </span>
                                        )}
                                    </TableCell>
                                    <TableCell className="text-muted-foreground text-xs">
                                        {new Date(job.created_at).toLocaleDateString()}
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-1">
                                            <Button variant="ghost" size="icon" asChild title="View">
                                                <Link to={`/jobs/${job.id}`}>
                                                    <Eye className="h-4 w-4" />
                                                </Link>
                                            </Button>

                                            {job.status === 'running' && (
                                                <Button
                                                    variant="ghost"
                                                    size="icon"
                                                    title="Pause"
                                                    onClick={() => pauseMutation.mutate(job.id)}
                                                    disabled={pauseMutation.isPending}
                                                >
                                                    <Pause className="h-4 w-4" />
                                                </Button>
                                            )}

                                            {job.status === 'paused' && (
                                                <Button
                                                    variant="ghost"
                                                    size="icon"
                                                    title="Resume"
                                                    onClick={() => resumeMutation.mutate(job.id)}
                                                    disabled={resumeMutation.isPending}
                                                >
                                                    <Play className="h-4 w-4" />
                                                </Button>
                                            )}

                                            {(job.status === 'pending' || job.status === 'running' || job.status === 'paused') && (
                                                <Button
                                                    variant="ghost"
                                                    size="icon"
                                                    title="Cancel"
                                                    onClick={() => {
                                                        if (confirm('Are you sure you want to cancel this job?')) {
                                                            cancelMutation.mutate(job.id)
                                                        }
                                                    }}
                                                    disabled={cancelMutation.isPending}
                                                >
                                                    <XCircle className="h-4 w-4" />
                                                </Button>
                                            )}

                                            {job.status === 'failed' && (
                                                <Button
                                                    variant="ghost"
                                                    size="icon"
                                                    title="Retry"
                                                    onClick={() => navigate('/jobs/new', { state: { cloneFrom: job, isRetry: true } })}
                                                >
                                                    <RotateCcw className="h-4 w-4" />
                                                </Button>
                                            )}

                                            <Button
                                                variant="ghost"
                                                size="icon"
                                                title="Clone"
                                                onClick={() => navigate('/jobs/new', { state: { cloneFrom: job } })}
                                            >
                                                <Copy className="h-4 w-4" />
                                            </Button>

                                            <Button
                                                variant="ghost"
                                                size="icon"
                                                className="text-destructive hover:text-destructive"
                                                title="Delete"
                                                onClick={() => {
                                                    if (confirm('Are you sure you want to delete this job?')) {
                                                        deleteMutation.mutate(job.id)
                                                    }
                                                }}
                                                disabled={deleteMutation.isPending}
                                            >
                                                <Trash2 className="h-4 w-4" />
                                            </Button>
                                        </div>
                                    </TableCell>
                                </TableRow>
                            ))
                        )}
                    </TableBody>
                </Table>
            </div>

            <div className="flex items-center justify-end space-x-2 py-4">
                <Button variant="outline" size="sm" disabled>Previous</Button>
                <Button variant="outline" size="sm" disabled>Next</Button>
            </div>
        </div>
    )
}
