import React from 'react'
import { useNavigate } from 'react-router-dom'
import { BackgroundLinesDemo } from '../components/BackgroundLinesDemo'
import { CoverDemo } from '../components/CoverDemo'
import LeaseLink from '../assets/LeaseLink.png'
import "../index.css";

export default function Landing() {
  const navigate = useNavigate();

  return (
    <div className="relative h-screen w-full bg-white">
      <BackgroundLinesDemo />
      <br></br>
      <div className="absolute top-0 left-0 right-0 bottom-0 flex flex-col items-center justify-center">
        <img
          src={LeaseLink}
          alt="LeaseLink"
          className="w-20 h-auto mb-6"
        />
        <h1 className="text-4xl font-bold text-indigo-700 mb-2">Welcome to LeaseLink</h1>
        <p className="mb-6 text-lg text-gray-600">Decentralized Rental Agreement Management</p>
        <button
          className="bg-indigo-600 text-white px-6 py-2 rounded-lg font-semibold shadow-lg hover:bg-indigo-700"
          onClick={() => navigate('/dashboard')}
        >
          Enter App
        </button>
      </div>
    </div>
  )
}
