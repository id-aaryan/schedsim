import subprocess
import matplotlib.pyplot as plt
import numpy as np

def process(mu):
    load = []
    performance = []
    lambdas = [(i + 1)*0.001 for i in range(100)]
    for l in lambdas:
        output = str(subprocess.check_output(f"./schedsim --topo=0 --mu={mu} --lambda={l} --genType=3 --procType=0", shell=True))
        load.append(l/mu)
        performance.append(float(output.split("\\t")[-2]))
    performance = [float(i) for i in performance]
    return load, performance

def display(mus):
    for mu in mus:
        load, performance = process(mu)
        xpoints = np.array(load)
        ypoints = np.array(performance)
        plt.plot(xpoints, ypoints)
        plt.show()

display([0.1])
