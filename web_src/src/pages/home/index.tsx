import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { superplaneListCanvases } from '../../api-client/sdk.gen'
import type { SuperplaneCanvas } from '../../api-client'

// Home page component - displays list of canvases
const HomePage = () => {
  const [canvases, setCanvases] = useState<SuperplaneCanvas[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  
  useEffect(() => {
    const fetchCanvases = async () => {
      try {
        setLoading(true)
        setError(null)
        
        // Call the API to fetch canvases
        const response = await superplaneListCanvases()
        
        if (response.data) {
          setCanvases(response.data.canvases || [])
        } else {
          throw new Error('No data received from API')
        }
      } catch (err) {
        console.error('Error fetching canvases:', err)
        setError('Failed to fetch canvases. Please try again later.')
      } finally {
        setLoading(false)
      }
    }
    
    fetchCanvases()
  }, [])

  return (
    <div className="flex justify-center w-full text-center">
      <div className="w-full max-w-6xl px-4 py-8">
        <h1 className="text-3xl font-bold mb-6">My Canvases</h1>
      
      {loading ? (
        <div className="flex justify-center items-center h-40">
          <p className="text-gray-500">Loading canvases...</p>
        </div>
      ) : error ? (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          <p>{error}</p>
        </div>
      ) : canvases.length === 0 ? (
        <div className="text-center py-8">
          <p className="text-gray-500 mb-4">No canvases found</p>
          <button className="flex items-center px-5 py-2.5 text-sm font-medium bg-indigo-600 text-white rounded-md hover:bg-indigo-700 transition-colors shadow-sm mx-auto">
            <svg className="mr-2 h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Create Your First Canvas
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {canvases.map((canvas) => (
            <div key={canvas.id} className="border rounded-lg p-5 hover:shadow-md transition-shadow flex flex-col">
              <h2 className="text-xl font-semibold mb-3">{canvas.name}</h2>
              <div className="mt-auto">
                <Link 
                  to={`canvas/${canvas.id}`}
                  className="inline-flex items-center text-sm font-medium text-indigo-600 hover:text-indigo-800 transition-colors group"
                >
                  <span>Open Canvas</span>
                  <svg className="ml-1 h-4 w-4 transform group-hover:translate-x-0.5 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 5l7 7m0 0l-7 7m7-7H3" />
                  </svg>
                </Link>
              </div>
            </div>
          ))}
        </div>
      )}
      </div>
    </div>
  )
}

export default HomePage