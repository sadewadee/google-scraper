import { useParams, Link, useNavigate } from "react-router-dom"
import { toast } from "sonner"
import { Button } from "@/components/UI/Button"
import { Badge } from "@/components/UI/Badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/UI/Card"
import { ArrowLeft, Clock, MapPin, Pause, Play, XCircle, Trash2, Copy, RotateCcw } from "lucide-react"
import { ResultsTable } from "@/components/Results/ResultsTable"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { jobsApi } from "@/api/jobs"

export default function JobDetail() {
    const { id } = useParams()
    const navigate = useNavigate()
    const queryClient = useQueryClient()

    const { data: jobRes, isLoading } = useQuery({
        queryKey: ["job", id],
        queryFn: () => jobsApi.getOne(id!),
        enabled: !!id,
        refetchInterval: (query) => {
            const job = query.state.data?.data
            return job?.status === 'running' || job?.status === 'pending' ? 3000 : false
        }
    })

    const pauseMutation = useMutation({
        mutationFn: jobsApi.pause,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["job", id] })
            toast.success("Job paused")
        },
        onError: () => {
            toast.error("Failed to pause job")
        }
    })

    const resumeMutation = useMutation({
        mutationFn: jobsApi.resume,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["job", id] })
            toast.success("Job resumed")
        },
        onError: () => {
            toast.error("Failed to resume job")
        }
    })

    const cancelMutation = useMutation({
        mutationFn: jobsApi.cancel,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["job", id] })
            toast.success("Job cancelled")
        },
        onError: () => {
            toast.error("Failed to cancel job")
        }
    })

    const deleteMutation = useMutation({
        mutationFn: jobsApi.delete,
        onSuccess: () => {
            toast.success("Job deleted")
            navigate("/jobs")
        },
        onError: () => {
            toast.error("Failed to delete job")
        }
    })

    const job = jobRes?.data

    if (isLoading) return <div className="p-8 text-center">Loading job details...</div>
    if (!job) return <div className="p-8 text-center">Job not found.</div>

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                    <Button variant="ghost" size="icon" asChild>
                        <Link to="/jobs">
                            <ArrowLeft className="h-4 w-4" />
                        </Link>
                    </Button>
                    <div>
                        <h2 className="text-2xl font-bold tracking-tight flex items-center gap-3">
                            Job #{id}
                            <Badge variant={job.status}>{job.status}</Badge>
                        </h2>
                        <p className="text-muted-foreground">Created on {new Date(job.created_at).toLocaleDateString()}</p>
                    </div>
                </div>
                <div className="flex items-center gap-2">
                    {job.status === 'failed' && (
                        <Button
                            variant="outline"
                            onClick={() => navigate('/jobs/new', { state: { cloneFrom: job, isRetry: true } })}
                        >
                            <RotateCcw className="h-4 w-4 mr-2" />
                            Retry
                        </Button>
                    )}
                    <Button
                        variant="outline"
                        onClick={() => navigate('/jobs/new', { state: { cloneFrom: job } })}
                    >
                        <Copy className="h-4 w-4 mr-2" />
                        Clone
                    </Button>
                    {job.status === 'running' && (
                        <Button
                            variant="outline"
                            onClick={() => pauseMutation.mutate(id!)}
                            disabled={pauseMutation.isPending}
                        >
                            <Pause className="h-4 w-4 mr-2" />
                            Pause
                        </Button>
                    )}
                    {job.status === 'paused' && (
                        <Button
                            variant="outline"
                            onClick={() => resumeMutation.mutate(id!)}
                            disabled={resumeMutation.isPending}
                        >
                            <Play className="h-4 w-4 mr-2" />
                            Resume
                        </Button>
                    )}
                    {(job.status === 'pending' || job.status === 'running' || job.status === 'paused') && (
                        <Button
                            variant="outline"
                            onClick={() => {
                                if (confirm('Are you sure you want to cancel this job?')) {
                                    cancelMutation.mutate(id!)
                                }
                            }}
                            disabled={cancelMutation.isPending}
                        >
                            <XCircle className="h-4 w-4 mr-2" />
                            Cancel
                        </Button>
                    )}
                    <Button
                        variant="destructive"
                        onClick={() => {
                            if (confirm('Are you sure you want to delete this job?')) {
                                deleteMutation.mutate(id!)
                            }
                        }}
                        disabled={deleteMutation.isPending}
                    >
                        <Trash2 className="h-4 w-4 mr-2" />
                        Delete
                    </Button>
                </div>
            </div>

            {/* Stats and Config */}
            <div className="grid gap-4 md:grid-cols-3">
                <Card>
                    <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Total Results</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">{job.progress.scraped_places}/{job.progress.total_places || "-"}</div>
                        {job.progress.percentage > 0 && (
                            <p className="text-xs text-muted-foreground mt-1">
                                {Math.round(job.progress.percentage)}% completed
                            </p>
                        )}
                    </CardContent>
                </Card>
                <Card>
                    <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Duration</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold flex items-center gap-2">
                            <Clock className="h-4 w-4 text-muted-foreground" />
                            -
                        </div>
                    </CardContent>
                </Card>
                <Card>
                    <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Configuration</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="text-sm space-y-1">
                            <div className="flex justify-between">
                                <span className="text-muted-foreground">Name:</span>
                                <span className="font-medium">{job.name}</span>
                            </div>
                            <div className="flex flex-col gap-1 mt-1">
                                <span className="text-muted-foreground">Keywords:</span>
                                <div className="flex flex-wrap gap-1">
                                    {job.config.keywords.map((k, i) => (
                                        <Badge key={i} variant="outline" className="text-xs">
                                            {k}
                                        </Badge>
                                    ))}
                                </div>
                            </div>
                            <div className="flex justify-between mt-2">
                                <span className="text-muted-foreground">Location:</span>
                                <span className="font-medium flex items-center gap-1">
                                    <MapPin className="h-3 w-3" />
                                    {job.config.geo_lat && job.config.geo_lon
                                        ? `${job.config.geo_lat}, ${job.config.geo_lon}`
                                        : "Auto"}
                                </span>
                            </div>
                        </div>
                    </CardContent>
                </Card>
            </div>

            {/* Results Table */}
            <Card>
                <CardHeader>
                    <CardTitle>Scraped Results</CardTitle>
                </CardHeader>
                <CardContent>
                    <ResultsTable jobId={id!} />
                </CardContent>
            </Card>
        </div>
    )
}
